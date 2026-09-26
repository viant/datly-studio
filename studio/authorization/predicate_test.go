package authorization

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/viant/scy/auth/jwt"
	xresponse "github.com/viant/xdatly/response"
	_ "modernc.org/sqlite"
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

func TestVersionMetadataAndSourceUseDistinctCapabilities(t *testing.T) {
	claims := &jwt.Claims{}
	claims.Subject = "viewer"
	input := &struct{ Jwt *jwt.Claims }{Jwt: claims}
	metadata, err := (&ReportVersionMetadataRead{InputBinding: InputBinding{Input: input}}).Compute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	source, err := (&ReportVersionRead{InputBinding: InputBinding{Input: input}}).Compute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(metadata.Expression, "can_view") || strings.Contains(metadata.Expression, "can_use_dql") ||
		!strings.Contains(source.Expression, "can_use_dql") ||
		!strings.Contains(metadata.Expression, "v.report_id") || len(metadata.Placeholders) != 2 {
		t.Fatalf("metadata=%+v source=%+v", metadata, source)
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
	if !strings.Contains(criteria.Expression, "can_publish") || !strings.Contains(criteria.Expression, "studio_auth_global.deleted_at IS NULL") ||
		len(criteria.Placeholders) != 2 || criteria.Placeholders[0] != "publisher" || criteria.Placeholders[1] != "publisher" {
		t.Fatalf("criteria = %+v", criteria)
	}
}

func TestGlobalPredicateMatchesLiveOwnerOrPublishGrant(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE reports (id TEXT PRIMARY KEY, owner_id TEXT NOT NULL, deleted_at TEXT)`,
		`CREATE TABLE report_acl (report_id TEXT, subject_type TEXT, subject_id TEXT, can_publish BOOLEAN)`,
		`CREATE TABLE authorization_predicates (name TEXT)`,
		`INSERT INTO reports(id,owner_id) VALUES ('r1','owner')`,
		`INSERT INTO report_acl VALUES ('r1','user','delegate',FALSE)`,
		`INSERT INTO authorization_predicates VALUES ('linked.handler')`,
	} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	visible := func(subject string) int {
		claims := &jwt.Claims{}
		claims.Subject = subject
		predicate := &AuthorizationPredicateRead{InputBinding: InputBinding{Input: &struct{ Jwt *jwt.Claims }{Jwt: claims}}}
		criteria, computeErr := predicate.Compute(ctx, nil)
		if computeErr != nil {
			t.Fatal(computeErr)
		}
		var count int
		if queryErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM authorization_predicates WHERE "+criteria.Expression, criteria.Placeholders...).Scan(&count); queryErr != nil {
			t.Fatal(queryErr)
		}
		return count
	}
	if visible("owner") != 1 || visible("delegate") != 0 {
		t.Fatal("global predicate did not distinguish owner from ungranted delegate")
	}
	if _, err = db.ExecContext(ctx, `UPDATE report_acl SET can_publish=TRUE WHERE subject_id='delegate'`); err != nil {
		t.Fatal(err)
	}
	if visible("delegate") != 1 {
		t.Fatal("live publish grant was not honored")
	}
	if _, err = db.ExecContext(ctx, `UPDATE reports SET deleted_at='2026-09-25' WHERE id='r1'`); err != nil {
		t.Fatal(err)
	}
	if visible("owner") != 0 || visible("delegate") != 0 {
		t.Fatal("deleted report retained global publish access")
	}
}

func TestPublicationEventReadRequiresCurrentOwner(t *testing.T) {
	claims := &jwt.Claims{}
	claims.Subject = "owner"
	predicate := &PublicationEventRead{InputBinding: InputBinding{Input: &struct{ Jwt *jwt.Claims }{Jwt: claims}}}
	criteria, err := predicate.Compute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(criteria.Expression, "studio_event_report.owner_id = ?") ||
		!strings.Contains(criteria.Expression, "studio_event_report.deleted_at IS NULL") ||
		len(criteria.Placeholders) != 1 || criteria.Placeholders[0] != "owner" {
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
