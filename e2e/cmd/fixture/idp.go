package main

// This test-only adapter models the public /v1/api/user/info wire contract.
// It has no dependency on the private Auth implementation. Policy evaluation
// remains in viant/authz; this adapter only verifies and supplies identity facts.
import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
)

var subjects = []string{"owner", "editor", "viewer", "runner", "outsider", "expired", "wrong-issuer", "sales-reader", "ops-reader", "no-exposure", "no-role", "customer-10", "customer-20", "customer-no-role", "customer-no-grant", "namespace-admin", "namespace-not-admin"}

func mockInfoURL() string { return fixtureURL(18795) + "/v1/api/user/info" }

func profileRoles(subject string) ([]string, []string) {
	switch subject {
	case "namespace-admin":
		return []string{"NAMESPACE_ADMIN"}, []string{}
	case "namespace-not-admin":
		return []string{"namespace_admin"}, []string{}
	case "owner":
		return []string{"ROLE_ADMIN", "ROLE_ANALYST", "XYZ"}, []string{"EXPOSE_SALES", "EXPOSE_OPERATIONS"}
	case "customer-10", "customer-20", "customer-no-grant":
		return []string{"XYZ"}, []string{}
	case "sales-reader":
		return []string{"ROLE_ANALYST"}, []string{"EXPOSE_SALES"}
	case "ops-reader":
		return []string{"ROLE_ANALYST"}, []string{"EXPOSE_OPERATIONS"}
	case "no-exposure":
		return []string{"ROLE_ANALYST"}, []string{}
	case "no-role":
		return []string{}, []string{"EXPOSE_SALES"}
	default:
		return []string{}, []string{}
	}
}
func profileEntityPermissions(subject string) []authz.EntityPermission {
	grants := []authz.EntityPermission{}
	add := func(id string) {
		grants = append(grants, authz.EntityPermission{Type: "customer", ID: authz.EntityID(id), Permissions: []string{"read"}})
	}
	switch subject {
	case "owner":
		add("10")
		add("20")
	case "customer-10", "customer-no-role":
		add("10")
	case "customer-20":
		add("20")
	}
	return grants
}
func prepareIDP(dir string) {
	base := filepath.Join(dir, "idp")
	check(os.MkdirAll(base, 0700))
	for i, subject := range subjects {
		token, err := os.ReadFile(filepath.Join(dir, subject+".token"))
		check(err)
		roles, features := profileRoles(subject)
		write(filepath.Join(base, fmt.Sprintf("%02d-req.json", i)), map[string]any{"Method": "GET", "URL": mockInfoURL(), "Header": map[string][]string{"Authorization": {"Bearer " + strings.TrimSpace(string(token))}}})
		body, err := json.Marshal(map[string]any{"status": "ok", "info": map[string]any{"subject": subject, "uid": strconv.Itoa(i + 1), "userId": i + 1, "accountId": 42, "roles": roles, "features": features, "entityPermissions": profileEntityPermissions(subject), "evaluatedAt": time.Now().UTC(), "validUntil": time.Now().Add(time.Hour).UTC()}})
		check(err)
		write(filepath.Join(base, fmt.Sprintf("%02d-resp.json", i)), map[string]any{"Code": 200, "Header": map[string][]string{"Content-Type": {"application/json"}}, "Body": string(body)})
	}
}

type idpProvider struct {
	config oauth.Config
	client *http.Client
}

func newIDP(dir string) (*idpProvider, error) {
	data, err := os.ReadFile(filepath.Join(dir, "public.pem"))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid fixture public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("fixture key is not RSA")
	}
	return &idpProvider{config: oauth.Config{Issuer: "studio-e2e", Audience: "studio-e2e", Algorithms: []string{"RS256"}, Keyfunc: func(*jwt.Token) (any, error) { return key, nil }}, client: &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (p *idpProvider) Resolve(ctx context.Context) (authz.Facts, error) {
	claims := struct {
		jwt.RegisteredClaims
		UserID    int `json:"user_id"`
		AccountID int `json:"account_id"`
	}{}
	bearer, err := oauth.VerifyToken(ctx, p.config, &claims)
	if err != nil {
		return authz.Facts{}, authz.ErrDenied
	}
	if claims.UserID < 1 || claims.AccountID != 42 || claims.ExpiresAt == nil {
		return authz.Facts{}, authz.ErrDenied
	}
	req, err := http.NewRequestWithContext(ctx, "GET", mockInfoURL(), nil)
	if err != nil {
		return authz.Facts{}, authz.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	response, err := p.client.Do(req)
	if err != nil {
		return authz.Facts{}, authz.ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return authz.Facts{}, authz.ErrUnavailable
	}
	var envelope struct {
		Status string `json:"status"`
		Info   *struct {
			Subject           string                    `json:"subject"`
			UserID            int                       `json:"userId"`
			AccountID         int                       `json:"accountId"`
			Roles             *[]string                 `json:"roles"`
			Features          *[]string                 `json:"features"`
			EntityPermissions *[]authz.EntityPermission `json:"entityPermissions"`
			EvaluatedAt       time.Time                 `json:"evaluatedAt"`
			ValidUntil        time.Time                 `json:"validUntil"`
		} `json:"info"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(&envelope); err != nil {
		return authz.Facts{}, authz.ErrUnavailable
	}
	info := envelope.Info
	now := time.Now()
	if envelope.Status != "ok" || info == nil || info.Subject != claims.Subject || info.UserID != claims.UserID || info.AccountID != claims.AccountID || info.Roles == nil || info.Features == nil || info.EntityPermissions == nil || info.EvaluatedAt.IsZero() || info.EvaluatedAt.After(now.Add(time.Second)) || !info.ValidUntil.After(now) {
		return authz.Facts{}, authz.ErrDenied
	}
	validUntil := info.ValidUntil
	if claims.ExpiresAt.Time.Before(validUntil) {
		validUntil = claims.ExpiresAt.Time
	}
	exposures := []string{}
	for _, feature := range *info.Features {
		if strings.HasPrefix(feature, "EXPOSE_") {
			exposures = append(exposures, feature)
		}
	}
	if err := authz.ValidateEntityPermissions(*info.EntityPermissions); err != nil {
		return authz.Facts{}, authz.ErrDenied
	}
	entities := []authz.Entity{}
	for _, grant := range *info.EntityPermissions {
		if authz.HasPermission(*info.EntityPermissions, grant.Type, grant.ID, "read") {
			entities = append(entities, authz.Entity{Type: grant.Type, ID: string(grant.ID)})
		}
	}
	return authz.Facts{Entities: entities, Subject: claims.Subject, Issuer: claims.Issuer, Tenant: "studio-e2e", Roles: *info.Roles, Exposures: exposures, EntityPermissions: *info.EntityPermissions, ValidUntil: validUntil}, nil
}
