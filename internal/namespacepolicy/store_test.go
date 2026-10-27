package namespacepolicy

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/authz"
	policystore "github.com/viant/authz/component/store/sql"
	"github.com/viant/datly-studio/internal/datatest"
	"testing"
	"time"
)

type factsProvider struct{ facts authz.Facts }

func (p *factsProvider) Resolve(context.Context) (authz.Facts, error) { return p.facts, nil }

func TestConfiguredAdministratorCanManageForeignNamespaceDefault(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "administrator_default", "studio")
	_, err := db.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,allowed_roles_json,etag,created_at,updated_at) VALUES('foreign','owner','foreign','Foreign','active','[]',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewReader(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close(ctx)
	facts := authz.Facts{Subject: "operator", Issuer: "idp", Tenant: "tenant", Roles: []string{"NAMESPACE_ADMIN"}, ValidUntil: time.Now().Add(time.Minute)}
	resource := Resource("foreign", "tenant")
	if allowed, err := reader.CanManage(ctx, facts, resource); err != nil || !allowed {
		t.Fatalf("configured administrator denied: %v", err)
	}
	facts.Roles = []string{"viewer"}
	if allowed, _ := reader.CanManage(ctx, facts, resource); allowed {
		t.Fatal("viewer can manage foreign namespace")
	}
	facts.Roles = []string{"NAMESPACE_ADMIN"}
	resource.Tenant = "other"
	if allowed, _ := reader.CanManage(ctx, facts, resource); allowed {
		t.Fatal("administrator crossed tenant")
	}
	resource.Tenant = "tenant"
	if _, err := db.ExecContext(ctx, `UPDATE namespaces SET status='archived' WHERE namespace_id='foreign'`); err != nil {
		t.Fatal(err)
	}
	if allowed, _ := reader.CanManage(ctx, facts, resource); allowed {
		t.Fatal("administrator managed archived namespace")
	}
}
func TestNamespaceDefaultPolicyPersistenceAndInheritance(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "defaults", "studio")
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"alpha", "beta"} {
		exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,allowed_roles_json,etag,created_at,updated_at) VALUES(?,'owner',?,?,'active','{"view":["reader"],"manage":["manager"]}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id, id, id)
	}
	exec(`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	for _, id := range []string{"one", "two", "foreign"} {
		ns := "alpha"
		if id == "foreign" {
			ns = "beta"
		}
		exec(`INSERT INTO components(id,slug,title,owner_id,status,namespace,namespace_id,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,'owner','active',?,?,'main',?,?,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id, id, id, ns, ns, id, id)
	}
	reader, err := NewReader(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close(ctx)
	raw := &policystore.Store{DB: db}
	defer raw.Close(ctx)
	provider := &factsProvider{authz.Facts{Subject: "delegate", Issuer: "test", Tenant: "tenant", Roles: []string{"manager"}, ValidUntil: time.Now().Add(time.Hour)}}
	admin := &authz.Administration{Store: raw, Provider: provider, EditorCheck: reader.CanManage}
	parent := Resource("alpha", "tenant")
	role := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	doc, err := admin.Create(ctx, authz.Document{Resource: parent, Policies: map[string]authz.Policy{"execute": role, "describe": role, "discover": role}})
	if err != nil {
		t.Fatal(err)
	}
	effective := &Store{Store: raw, Namespace: reader.Namespace}
	service := &authz.Service{Store: effective, Provider: provider}
	provider.facts.Roles = []string{"reader"}
	resource := authz.Resource{Kind: "component", ID: "one", Version: "1", Tenant: "tenant"}
	for _, id := range []string{"one", "two"} {
		resource.ID = id
		for _, action := range []string{"discover", "describe", "execute"} {
			if _, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: action}); err != nil {
				t.Fatalf("%s %s: %v", id, action, err)
			}
		}
	}
	for _, other := range []authz.Resource{{Kind: "component", ID: "foreign", Version: "1", Tenant: "tenant"}, {Kind: "component", ID: "one", Version: "1", Tenant: "other"}, {Kind: "skill", ID: "one", Version: "1", Tenant: "tenant"}} {
		if _, err := service.Authorize(ctx, authz.Request{Resource: other, Action: "execute"}); err == nil {
			t.Fatal("default escaped namespace/tenant/kind")
		}
	}
	resource.ID = "one"
	if _, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: "manageAccess"}); err == nil {
		t.Fatal("inherited management")
	}
	provider.facts.Roles = nil
	if _, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("role revocation not applied")
	}
	provider.facts.Roles = []string{"manager"}
	doc.Policies["execute"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	doc, err = admin.Replace(ctx, doc)
	if err != nil {
		t.Fatal(err)
	}
	provider.facts.Roles = []string{"reader"}
	provider.facts.Entities = []authz.Entity{{Type: "project", ID: "p1"}}
	decision, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: "execute"})
	if err != nil || !decision.Bounded || len(decision.Entities) != 1 {
		t.Fatalf("lost namespace entity bounds: %+v %v", decision, err)
	}
	_, err = raw.Provision(ctx, authz.Document{Resource: resource, Policies: map[string]authz.Policy{"execute": {Mode: "deny"}}}, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("default broadened explicit deny")
	}
	if _, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: "discover"}); err == nil {
		t.Fatal("default filled missing action in explicit document")
	}
	// Delegate authority is current namespace state, not a rule saved in the policy.
	provider.facts.Roles = []string{"manager"}
	exec(`UPDATE namespaces SET allowed_roles_json='[]' WHERE namespace_id='alpha'`)
	if _, err = admin.Replace(ctx, doc); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("revoked manager saved default: %v", err)
	}
	provider.facts.Subject = "owner"
	if _, err = admin.Replace(ctx, doc); err != nil {
		t.Fatalf("owner cannot manage default: %v", err)
	}
	exec(`UPDATE namespaces SET status='archived' WHERE namespace_id='alpha'`)
	if _, err = effective.Get(ctx, authz.Resource{Kind: "component", ID: "two", Version: "1", Tenant: "tenant"}); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("inactive namespace falls through: %v", err)
	}
}

type failingStore struct {
	authz.Store
	err error
}

func (s failingStore) Get(context.Context, authz.Resource) (authz.Document, error) {
	return authz.Document{}, s.err
}
func TestNamespaceDefaultDoesNotHideStoreFailures(t *testing.T) {
	s := &Store{Store: failingStore{err: sql.ErrConnDone}, Namespace: func(context.Context, string) (string, error) { t.Fatal("lookup after store error"); return "", nil }}
	_, err := s.Get(context.Background(), authz.Resource{Kind: "component", ID: "x"})
	if !errors.Is(err, sql.ErrConnDone) {
		t.Fatal(err)
	}
}
