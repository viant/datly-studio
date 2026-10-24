package usegrant_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/datly-studio/sdk/artifactrelease/usegrant"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

type factsProvider struct{ role string }

func (p factsProvider) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: "recipient", Tenant: "tenant", Issuer: "tenant", Roles: []string{p.role}, ValidUntil: time.Now().Add(time.Hour)}, nil
}

func TestIndependentSDKReaderGrantsOnlyActiveExactComponent(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	scope := sqlstore.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo"}
	producer, err := sqlstore.New(db, scope, "verified-publisher")
	if err != nil {
		t.Fatal(err)
	}
	api, err := sqlstore.New(db, scope, "sdk-service")
	if err != nil {
		t.Fatal(err)
	}
	root := artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1, Hash: strings.Repeat("a", 64)}
	bundle := contract.Bundle{Namespace: "demo", Roots: []artifactcohort.RootRef{root}}
	bundle.Hash = contract.BundleHash(bundle)
	policy := authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}
	ref := authz.NamedReference{Resource: policy, Revision: 1}
	doc := authz.Document{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Operators", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
	}}}}}
	key := contract.Key{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo", Name: "main"}
	channel, err := contract.NewChannel(key, "", []authz.NamedReference{ref})
	if err != nil {
		t.Fatal(err)
	}
	if err = producer.PutBundle(ctx, bundle); err != nil {
		t.Fatal(err)
	}
	if err = producer.CreateChannel(ctx, channel); err != nil {
		t.Fatal(err)
	}
	next, err := contract.PreparePromotion(channel, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := contract.CompileCohorts(ctx, next, bundle, []authz.Document{doc})
	if err != nil {
		t.Fatal(err)
	}
	binding := sqlstore.ComponentBinding{Root: root, DataSourceID: "rows", ComponentKind: "dynamic", ComponentID: "reader", ComponentRevision: "7", Method: http.MethodGet, Route: "/rows", SchemaFingerprint: strings.Repeat("b", 64), ContentFingerprint: strings.Repeat("c", 64)}
	id, err := producer.StageRevisionWithBindingsAndDependencies(ctx, channel, next, bundle, []sqlstore.ComponentBinding{binding}, compiled.Dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if err = producer.CompleteRevision(ctx, key, 0, id); err != nil {
		t.Fatal(err)
	}
	provider := factsProvider{role: "operator"}
	reader := &usegrant.Reader{Store: api, Scope: scope, VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil }, Provider: provider,
		NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
			return []authz.Document{doc}, nil
		},
		FenceDependencies: func(_ context.Context, deps []authz.NamedDependency) error {
			if !reflect.DeepEqual(deps, compiled.Dependencies) {
				return contract.ErrConflict
			}
			return nil
		},
		RootAvailable: func(context.Context, artifactcohort.RootRef) error { return nil },
		NativeConstraints: func(_ context.Context, b sqlstore.ComponentBinding, _ string) error {
			if b.ComponentID != "reader" || b.ComponentRevision != "7" {
				return authz.ErrDenied
			}
			return nil
		},
	}
	allowed, err := reader.Allowed(ctx, "reader", "7", "run")
	if err != nil || !allowed {
		t.Fatalf("released native reader denied: %t %v", allowed, err)
	}
	for _, query := range []struct{ id, revision, permission string }{{"reader", "8", "run"}, {"unrelated", "7", "run"}, {"reader", "7", "edit"}} {
		allowed, err = reader.Allowed(ctx, query.id, query.revision, query.permission)
		if query.permission == "edit" {
			if err == nil {
				t.Fatal("edit grant accepted")
			}
			continue
		}
		if err != nil || allowed {
			t.Fatalf("unrelated pin admitted: %+v allowed=%t err=%v", query, allowed, err)
		}
	}
	reader.Provider = factsProvider{role: "viewer"}
	allowed, err = reader.Allowed(ctx, "reader", "7", "run")
	if err != nil || allowed {
		t.Fatalf("unassigned actor admitted: %t %v", allowed, err)
	}
	reader.Provider = provider
	// Move the release during the final dependency check. The named policy
	// revision is unchanged, so only a channel fence can reject the old grant.
	fenceCalls := 0
	reader.FenceDependencies = func(context.Context, []authz.NamedDependency) error {
		fenceCalls++
		if fenceCalls != 2 {
			return nil
		}
		changedRoot := root
		changedRoot.Version = 2
		changedRoot.Hash = strings.Repeat("d", 64)
		changedBundle := contract.Bundle{Namespace: "demo", Roots: []artifactcohort.RootRef{changedRoot}}
		changedBundle.Hash = contract.BundleHash(changedBundle)
		if e := producer.PutBundle(ctx, changedBundle); e != nil {
			t.Fatal(e)
		}
		changed, e := contract.PreparePromotion(next, next.Revision, changedBundle)
		if e != nil {
			t.Fatal(e)
		}
		changedBinding := binding
		changedBinding.Root = changedRoot
		changedBinding.ComponentRevision = "8"
		op, e := producer.StageRevisionWithBindingsAndDependencies(ctx, next, changed, changedBundle, []sqlstore.ComponentBinding{changedBinding}, compiled.Dependencies)
		if e != nil {
			t.Fatal(e)
		}
		if e = producer.CompleteRevision(ctx, key, next.Revision, op); e != nil {
			t.Fatal(e)
		}
		return nil
	}
	allowed, err = reader.Allowed(ctx, "reader", "7", "run")
	if allowed || !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("moved channel admitted old component: %t %v", allowed, err)
	}
	reader.FenceDependencies = func(context.Context, []authz.NamedDependency) error { return nil }
	allowed, err = reader.Allowed(ctx, "reader", "7", "run")
	if err != nil || allowed {
		t.Fatalf("old component remains active: %t %v", allowed, err)
	}
	// Restore the original release for the independent policy-head test.
	current, e := producer.Channel(ctx, key)
	if e != nil {
		t.Fatal(e)
	}
	restored, e := contract.PreparePromotion(current, current.Revision, bundle)
	if e != nil {
		t.Fatal(e)
	}
	op, e := producer.StageRevisionWithBindingsAndDependencies(ctx, current, restored, bundle, []sqlstore.ComponentBinding{binding}, compiled.Dependencies)
	if e != nil {
		t.Fatal(e)
	}
	if e = producer.CompleteRevision(ctx, key, current.Revision, op); e != nil {
		t.Fatal(e)
	}
	reader.FenceDependencies = func(context.Context, []authz.NamedDependency) error { return contract.ErrConflict }
	allowed, err = reader.Allowed(ctx, "reader", "7", "run")
	if allowed || err != nil {
		t.Fatalf("changed policy head admitted: %t %v", allowed, err)
	}
}
