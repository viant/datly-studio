package authorization

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/viant/scy/auth/jwt"
	xresponse "github.com/viant/xdatly/response"
)

func TestReportReadBuildsOwnerAndACLAuthorizationCriteria(t *testing.T) {
	claims := &jwt.Claims{}
	claims.Subject = "viewer"
	predicate := &ReportRead{InputBinding: InputBinding{Input: &struct{ Jwt *jwt.Claims }{Jwt: claims}}}
	criteria, err := predicate.Compute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(criteria.Expression, "reports.id") || !strings.Contains(criteria.Expression, "can_view") {
		t.Fatalf("criteria = %+v", criteria)
	}
	if len(criteria.Placeholders) != 2 || criteria.Placeholders[0] != "viewer" || criteria.Placeholders[1] != "viewer" {
		t.Fatalf("placeholders = %#v", criteria.Placeholders)
	}
}

func TestAuthorizationDoesNotUseDisplayClaimsAsIdentity(t *testing.T) {
	claims := &jwt.Claims{Username: "viewer", Email: "viewer@example.com"}
	predicate := &ReportRead{InputBinding: InputBinding{Input: &struct{ Jwt *jwt.Claims }{Jwt: claims}}}
	_, err := predicate.Compute(context.Background(), nil)
	var actual *xresponse.Error
	if !errors.As(err, &actual) || actual.Code != 403 {
		t.Fatalf("error = %v, want JWT sub authorization error", err)
	}
}

func TestAuthorizationRequiresBoundVerifiedClaims(t *testing.T) {
	_, err := (&ReportRead{InputBinding: InputBinding{Input: &struct{ Jwt *jwt.Claims }{}}}).Compute(context.Background(), nil)
	var actual *xresponse.Error
	if !errors.As(err, &actual) || actual.Code != 403 {
		t.Fatalf("error = %v, want 403 authorization error", err)
	}
}

func TestRuntimeAuthorizationRequiresPublishPermission(t *testing.T) {
	claims := &jwt.Claims{}
	claims.Subject = "publisher"
	predicate := &RuntimeRead{InputBinding: InputBinding{Input: &struct{ Jwt *jwt.Claims }{Jwt: claims}}}
	criteria, err := predicate.Compute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(criteria.Expression, "can_publish") || len(criteria.Placeholders) != 1 || criteria.Placeholders[0] != "publisher" {
		t.Fatalf("criteria = %+v", criteria)
	}
}

func TestReportEditRejectsCrossOwnerCreate(t *testing.T) {
	claims := &jwt.Claims{}
	claims.Subject = "owner-a"
	input := &struct {
		Jwt     *jwt.Claims
		Reports []struct{ OwnerId string }
	}{Jwt: claims, Reports: []struct{ OwnerId string }{{OwnerId: "owner-b"}}}
	_, err := (&ReportEdit{InputBinding: InputBinding{Input: input}}).Compute(context.Background(), nil)
	var actual *xresponse.Error
	if !errors.As(err, &actual) || actual.Code != 403 {
		t.Fatalf("error = %v, want 403 authorization error", err)
	}
}
