package artifactrelease_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

func TestPolicyRefreshWithdrawsOnlyAffectedChannel(t *testing.T) {
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
	store, err := sqlstore.New(db, scope, "trusted-producer")
	if err != nil {
		t.Fatal(err)
	}
	publisher := artifactcohort.NewPublisher()
	if _, err = publisher.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	artifacts := map[string]*artifactcohort.Artifact{}
	for _, name := range []string{"a", "b"} {
		payload := json.RawMessage(`{"name":"` + name + `"}`)
		ref := artifactcohort.RootRef{URI: "window://demo/" + name, Kind: "window", Version: 1, Hash: artifactcohort.Hash(payload)}
		artifacts[ref.URI] = &artifactcohort.Artifact{Root: ref, Payload: payload}
	}
	docFor := func(name string) authz.Document {
		resource := authz.Resource{Tenant: "tenant", Kind: "policy", ID: name, Version: "logical"}
		return authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: name, Policies: map[string]authz.Policy{
			"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: name}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: name}},
		}}}}}
	}
	docs := map[string]authz.Document{"a": docFor("a"), "b": docFor("b")}
	newController := func(p *artifactcohort.Publisher) *sdkrelease.Controller {
		controller, e := sdkrelease.NewController(sdkrelease.ControllerConfig{Scope: scope, Store: store, Publisher: p,
			VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil }, Authorize: func(context.Context, string) error { return nil },
			Candidates: func(context.Context) ([]contract.Candidate, error) { return nil, nil },
			Prepare: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				out := make([]*artifactcohort.Artifact, 0, len(refs))
				for _, ref := range refs {
					out = append(out, artifacts[ref.URI])
				}
				return out, nil
			},
			RestoreArtifacts: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				out := make([]*artifactcohort.Artifact, 0, len(refs))
				for _, ref := range refs {
					out = append(out, artifacts[ref.URI])
				}
				return out, nil
			},
			AvailableRoot: func(context.Context, artifactcohort.RootRef) error { return nil }, ValidateArtifact: func(context.Context, *artifactcohort.Artifact) error { return nil },
			CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
				return nil, nil
			},
			NamedDocuments: func(_ context.Context, refs []authz.NamedReference) ([]authz.Document, error) {
				out := make([]authz.Document, 0, len(refs))
				for _, ref := range refs {
					out = append(out, docs[ref.Resource.ID])
				}
				return out, nil
			},
			Fence: func(_ context.Context, deps []authz.NamedDependency) error {
				for _, dep := range deps {
					if docs[dep.Resource.ID].Revision != dep.Revision {
						return contract.ErrConflict
					}
				}
				return nil
			},
			AdmitAudience: func(context.Context, contract.Channel, authz.NamedReference) (bool, error) { return true, nil },
		})
		if e != nil {
			t.Fatal(e)
		}
		return controller
	}
	controller := newController(publisher)
	if err = controller.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		ref := authz.NamedReference{Resource: docs[name].Resource, Revision: 1}
		if _, err = controller.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: name, Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{artifacts["window://demo/"+name].Root}}); err != nil {
			t.Fatal(err)
		}
	}
	bad := docs["a"]
	bad.Revision = 2
	bad.Policies[authz.NamedDefinitionAction].Definition.Archived = true
	docs["a"] = bad
	if err = controller.RefreshPolicies(ctx, bad.Resource); err == nil {
		t.Fatal("archived named head accepted")
	}
	snapshot, err := controller.ActiveSnapshot(ctx)
	if err != nil || len(snapshot.Selections) != 1 || snapshot.Selections[0].Channel.Key.Name != "b" {
		t.Fatalf("unrelated channel was blocked: %+v %v", snapshot, err)
	}
	fresh := artifactcohort.NewPublisher()
	if _, err = fresh.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	restarted := newController(fresh)
	if err = restarted.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, err = restarted.ActiveSnapshot(ctx)
	if err != nil || len(snapshot.Selections) != 1 || snapshot.Selections[0].Channel.Key.Name != "b" {
		t.Fatalf("restart restored archived channel or dropped unrelated: %+v %v", snapshot, err)
	}
}
