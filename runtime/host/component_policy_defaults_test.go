package host

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"

	accessstore "github.com/viant/authz/datly/store/sql"
	"github.com/viant/datly-studio/schema"
	"testing"

	"github.com/viant/authz"
)

type defaultPolicyFixture struct {
	document authz.Document
	err      error
}

func (s *defaultPolicyFixture) Get(context.Context, authz.Resource) (authz.Document, error) {
	return s.document, s.err
}
func (s *defaultPolicyFixture) Replace(context.Context, authz.Document, int64, string) (authz.Document, error) {
	return authz.Document{}, errors.New("administration is not used")
}

func TestPublishedComponentDefaultsCannotBroadenExplicitOrUnknownPolicies(t *testing.T) {
	resource := authz.Resource{Kind: "component", ID: "known", Version: "2", Tenant: "*"}
	backend := &defaultPolicyFixture{err: sql.ErrNoRows}
	store := &componentPolicyDefaults{Store: backend, tenant: "*", versions: map[string]int{"known": 2}}
	service := &authz.Service{Store: store}
	for _, action := range []string{"discover", "describe", "execute"} {
		if _, err := service.Authorize(context.Background(), authz.Request{Resource: resource, Action: action}); err != nil {
			t.Fatalf("public %s denied: %v", action, err)
		}
	}
	if _, err := service.Authorize(context.Background(), authz.Request{Resource: resource, Action: "modify_policy"}); err == nil {
		t.Fatal("default granted policy administration")
	}
	for _, other := range []authz.Resource{
		{Kind: "component", ID: "unknown", Version: "2", Tenant: "*"},
		{Kind: "component", ID: "known", Version: "1", Tenant: "*"},
		{Kind: "component", ID: "known", Version: "2", Tenant: "other"},
		{Kind: "skill", ID: "known", Version: "2", Tenant: "*"},
	} {
		if _, err := service.Authorize(context.Background(), authz.Request{Resource: other, Action: "execute"}); err == nil {
			t.Fatalf("unknown identity received default: %+v", other)
		}
	}
	backend.err = sql.ErrConnDone
	if _, err := service.Authorize(context.Background(), authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("database failure became public")
	}
	backend.err = nil
	backend.document = authz.Document{Resource: resource, Revision: 4, Policies: map[string]authz.Policy{"describe": {Mode: "public"}}}
	if _, err := service.Authorize(context.Background(), authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("explicit missing action became public")
	}
	backend.document.Policies["execute"] = authz.Policy{Mode: "protected"}
	if _, err := service.Authorize(context.Background(), authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("explicit protected policy became public")
	}
	backend.err = sql.ErrNoRows
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Authorize(ctx, authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("cancelled request received default")
	}
}

func TestPublishedDefaultsPinSourceVersions(t *testing.T) {
	source := map[string]int{"known": 2}
	parent := &Service{config: Config{Access: &ResourceAccessConfig{Tenant: "*"}}, resourceAccess: &authz.Service{Store: &defaultPolicyFixture{err: sql.ErrNoRows}}}
	pinned := parent.publishedAuthorizer(source)
	source["foreign"] = 1
	source["known"] = 3
	if _, err := pinned.resourceAccess.Authorize(context.Background(), authz.Request{Resource: authz.Resource{Kind: "component", ID: "known", Version: "2", Tenant: "*"}, Action: "execute"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pinned.resourceAccess.Authorize(context.Background(), authz.Request{Resource: authz.Resource{Kind: "component", ID: "foreign", Version: "1", Tenant: "*"}, Action: "execute"}); err == nil {
		t.Fatal("later map change expanded published defaults")
	}
}

func TestPublicDefaultsRejectAStoredHeadWithMissingRevision(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	backend := &accessstore.Store{DB: db}
	defer backend.Close(ctx)
	resource := authz.Resource{Kind: "component", ID: "known", Version: "2", Tenant: "*"}
	if _, err = backend.Provision(ctx, authz.Document{Resource: resource, Policies: map[string]authz.Policy{"describe": {Mode: "public"}}}, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM resource_policy_revisions"); err != nil {
		t.Fatal(err)
	}
	service := &authz.Service{Store: &componentPolicyDefaults{Store: backend, tenant: "*", versions: map[string]int{"known": 2}}}
	if _, err = service.Authorize(ctx, authz.Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("damaged explicit policy received public defaults")
	}
}
