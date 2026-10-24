package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/releasegate"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

func TestArtifactReleaseCatalogServingUsesActivationGate(t *testing.T) {
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
	store, err := sqlstore.New(db, scope, "verified-editor")
	if err != nil {
		t.Fatal(err)
	}
	publisher := artifactcohort.NewPublisher()
	if _, err = publisher.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	catalog := NewPublishedArtifactCatalog(publisher)
	policy := authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}
	ref := authz.NamedReference{Resource: policy, Revision: 1}
	doc := authz.Document{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Operators", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
	}}}}}
	body := json.RawMessage(`{"title":"Released"}`)
	root := artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1, Hash: artifactcohort.Hash(body)}
	controller, err := sdkrelease.NewController(sdkrelease.ControllerConfig{
		Scope: scope, Store: store, Publisher: publisher,
		VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil },
		Authorize:   func(context.Context, string) error { return nil },
		Candidates:  func(context.Context) ([]contract.Candidate, error) { return []contract.Candidate{{Root: root}}, nil },
		Prepare: func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
			return []*artifactcohort.Artifact{{Root: root, Payload: body}}, nil
		},
		CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
			return nil, nil
		},
		RestoreArtifacts: func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
			return []*artifactcohort.Artifact{{Root: root, Payload: body}}, nil
		},
		ValidateArtifact: func(context.Context, *artifactcohort.Artifact) error { return nil },
		NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
			return []authz.Document{doc}, nil
		},
		Fence:         func(context.Context, []authz.NamedDependency) error { return nil },
		AdmitAudience: func(context.Context, contract.Channel, authz.NamedReference) (bool, error) { return true, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = catalog.BindReleaseGate(controller.Gate()); err != nil {
		t.Fatal(err)
	}
	if _, err = catalog.Discover(ctx); !errors.Is(err, releasegate.ErrNotReady) {
		t.Fatalf("catalog served before reconciliation: %v", err)
	}
	if err = controller.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = controller.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: "main", Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{root}}); err != nil {
		t.Fatal(err)
	}
	manifest, err := catalog.Discover(ctx)
	if err != nil || len(manifest.Cohorts) != 1 || manifest.Cohorts[0].Roots[0] != root {
		t.Fatalf("release not visible after complete: %+v %v", manifest.Cohorts, err)
	}
	artifact, err := catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "tenant", Root: root})
	if err != nil || artifact.Root != root {
		t.Fatalf("exact released artifact unavailable: %+v %v", artifact, err)
	}
	other := root
	other.Hash = strings.Repeat("b", 64)
	if _, err = catalog.Fetch(ctx, artifactcohort.FetchInput{Tenant: "tenant", Root: other}); !errors.Is(err, artifactcohort.ErrDenied) {
		t.Fatalf("unadvertised root served: %v", err)
	}
}
