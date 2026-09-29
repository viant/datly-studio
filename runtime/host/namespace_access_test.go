package host

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	_ "modernc.org/sqlite"
)

type namespaceFacts struct{ value authz.Facts }

func (p namespaceFacts) Resolve(context.Context) (authz.Facts, error) { return p.value, nil }

func TestRuntimeNamespaceGateProtectsDiscoveryAndExecutionVisibility(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	id := namespaceaccess.ID("owner", "private")
	if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,allowed_roles_json,etag,created_at,updated_at) VALUES(?,'owner','private','Private','active','private','["analyst"]',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id); err != nil {
		t.Fatal(err)
	}
	service := &Service{studio: db, config: Config{NamespaceID: id, Authentication: Authentication{DefaultMode: "public"}}}
	if err := service.authorizeNamespace(ctx); err == nil {
		t.Fatal("public component mode bypassed private namespace")
	}
	claims := &jwt.Claims{RegisteredClaims: jwtlib.RegisteredClaims{Subject: "owner", ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Minute))}}
	owner := context.WithValue(ctx, verifiedClaimsKey{}, claims)
	if err := service.authorizeNamespace(owner); err != nil {
		t.Fatalf("verified owner denied: %v", err)
	}
	key := spec.Key{Kind: spec.KindComponent, Scope: "fixture", Name: "reader"}
	tool, resource, _ := service.catalogAuthorizers(map[spec.Key]string{key: "component"}, map[string]int{"component": 1}, map[string]string{"skill://private/": "component"})
	if err := tool(ctx, dexec.ComponentTarget{Component: key}, "describe"); err == nil {
		t.Fatal("tool discovery bypassed private namespace")
	}
	if err := resource(ctx, "skill://private/SKILL.md", "retrieve"); err == nil {
		t.Fatal("skill read bypassed private namespace")
	}
	service.resourceAccess = &authz.Service{Provider: namespaceFacts{authz.Facts{Subject: "viewer", Issuer: "verified", ValidUntil: time.Now().Add(time.Minute), Roles: []string{"analyst"}}}}
	if err := service.authorizeNamespace(ctx); err != nil {
		t.Fatalf("assigned role denied: %v", err)
	}
	if _, err := db.Exec(`UPDATE namespaces SET allowed_roles_json='[]' WHERE namespace_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := service.authorizeNamespace(ctx); err == nil {
		t.Fatal("revoked role remained authorized")
	}
	if _, err := db.Exec(`UPDATE namespaces SET visibility='public' WHERE namespace_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := service.authorizeNamespace(ctx); err != nil {
		t.Fatalf("public namespace denied: %v", err)
	}
	if _, err := db.Exec(`UPDATE namespaces SET status='archived' WHERE namespace_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := service.authorizeNamespace(owner); err == nil {
		t.Fatal("archived namespace owner remained authorized")
	}
}
