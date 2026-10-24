package artifactrelease_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

type audienceRoleKey struct{}

func TestControllerPublishesExactBundleAndRollsBackWithoutRecompiling(t *testing.T) {
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
	resource := authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}
	ref := authz.NamedReference{Resource: resource, Revision: 1}
	doc := authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Operators", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
	}}}}}
	prepared := 0
	restored := 0
	allowPublish := true
	revokeDuringPrepare := false
	policyHeadRevision := int64(1)
	makeArtifacts := func(refs []artifactcohort.RootRef) []*artifactcohort.Artifact {
		result := make([]*artifactcohort.Artifact, 0, len(refs))
		for _, root := range refs {
			body := json.RawMessage(`{"version":1}`)
			if root.Version == 2 {
				body = json.RawMessage(`{"version":2}`)
			}
			result = append(result, &artifactcohort.Artifact{Root: root, Payload: body})
		}
		return result
	}
	controller, err := sdkrelease.NewController(sdkrelease.ControllerConfig{
		Scope: scope, Store: store, Publisher: publisher,
		VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil },
		Authorize: func(_ context.Context, action string) error {
			if action == "publish" && !allowPublish {
				return errors.New("editor revoked")
			}
			if action == "publish" || action == "promote" || action == "list" || action == "history" || action == "candidates" || action == "use" {
				return nil
			}
			return errors.New("denied")
		},
		Candidates: func(context.Context) ([]contract.Candidate, error) { return nil, nil },
		Prepare: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
			prepared++
			if revokeDuringPrepare {
				allowPublish = false
			}
			return makeArtifacts(refs), nil
		},
		CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
			return nil, nil
		},
		RestoreArtifacts: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
			restored++
			return makeArtifacts(refs), nil
		},
		ValidateArtifact: func(context.Context, *artifactcohort.Artifact) error { return nil },
		NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
			return []authz.Document{doc}, nil
		},
		Fence: func(_ context.Context, deps []authz.NamedDependency) error {
			for _, dep := range deps {
				if dep.Revision != policyHeadRevision {
					return contract.ErrConflict
				}
			}
			return nil
		},
		AdmitAudience: func(ctx context.Context, _ contract.Channel, _ authz.NamedReference) (bool, error) {
			role, _ := ctx.Value(audienceRoleKey{}).(string)
			return role == "operator", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = controller.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	root := func(version uint64) artifactcohort.RootRef {
		body := json.RawMessage(`{"version":1}`)
		if version == 2 {
			body = json.RawMessage(`{"version":2}`)
		}
		return artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: version, Hash: artifactcohort.Hash(body)}
	}
	first, err := controller.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: "main", Description: "Main", Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{root(1)}, ExpectedRevision: 0})
	if err != nil || !first.Activated || first.Channel.Revision != 1 {
		t.Fatalf("first publish: %+v %v", first, err)
	}
	firstSnapshot, err := controller.ActiveSnapshot(ctx)
	if err != nil || len(firstSnapshot.Selections) != 1 || firstSnapshot.Selections[0].Bundle.Hash != first.Bundle.Hash {
		t.Fatalf("active selection: %+v %v", firstSnapshot, err)
	}
	operator := context.WithValue(ctx, audienceRoleKey{}, "operator")
	if _, err = controller.AdmitRoot(ctx, root(1)); err == nil {
		t.Fatal("unassigned actor admitted")
	}
	admittedHash, err := controller.AdmitRoot(operator, root(1))
	if err != nil || admittedHash != firstSnapshot.ManifestHash {
		t.Fatalf("operator admission: %s %v", admittedHash, err)
	}
	second, err := controller.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: "main", Description: "Main", Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{root(2)}, ExpectedRevision: 1})
	if err != nil || !second.Activated || second.Channel.Revision != 2 || prepared != 2 {
		t.Fatalf("second publish: %+v %v prepares=%d", second, err, prepared)
	}
	if err = controller.FenceSnapshot(ctx, firstSnapshot.ManifestHash); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("stale business release not fenced: %v", err)
	}
	if err = controller.FenceRoot(operator, root(1), admittedHash); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("stale root result not fenced: %v", err)
	}
	manifest, err := publisher.Snapshot()
	if err != nil || len(manifest.Cohorts) != 1 || len(manifest.Cohorts[0].Roots) != 1 || manifest.Cohorts[0].Roots[0] != root(2) {
		t.Fatalf("second revision not selected: %+v %v", manifest.Cohorts, err)
	}
	rollback, err := controller.Promote(ctx, contract.PromoteInput{Namespace: "demo", Name: "main", BundleHash: first.Bundle.Hash, ExpectedRevision: 2})
	if err != nil || rollback.Channel.Revision != 3 || prepared != 2 {
		t.Fatalf("rollback rebuilt source: %+v %v prepares=%d", rollback, err, prepared)
	}
	manifest, err = publisher.Snapshot()
	if err != nil || manifest.Cohorts[0].Roots[0] != root(1) {
		t.Fatalf("rollback did not select exact retained revision: %+v %v", manifest.Cohorts, err)
	}
	listed, err := controller.List(ctx, contract.ListInput{Namespace: "demo"})
	if err != nil || len(listed.Channels) != 1 || listed.Channels[0].Channel.Revision != 3 {
		t.Fatalf("list: %+v %v", listed, err)
	}
	policyHeadRevision = 2
	if _, err = controller.ActiveSnapshot(ctx); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("stale named policy head admitted: %v", err)
	}
	policyHeadRevision = 1
	history, err := controller.History(ctx, contract.HistoryInput{Namespace: "demo", Name: "main"})
	if err != nil || len(history.Entries) != 3 {
		t.Fatalf("history: %+v %v", history, err)
	}
	revokeDuringPrepare = true
	if _, err = controller.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: "main", Description: "Unauthorized", Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{root(2)}, ExpectedRevision: 3}); err == nil {
		t.Fatal("revoked editor activated after long preparation")
	}
	committed, err := store.Channel(ctx, rollback.Channel.Key)
	if err != nil || committed.Revision != 3 || committed.ActiveBundleHash != first.Bundle.Hash {
		t.Fatalf("revoked operation changed active channel: %+v %v", committed, err)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("revoked operation staged durable intent: %+v %v", pending, err)
	}
	if restored != 0 {
		t.Fatalf("warm producer reloaded bytes unnecessarily: %d", restored)
	}
	// A fresh process starts with no retained versions. It restores the exact
	// active bundle from producer persistence without recompiling source.
	freshPublisher := artifactcohort.NewPublisher()
	if _, err = freshPublisher.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
		t.Fatal(err)
	}
	restarted, err := sdkrelease.NewController(sdkrelease.ControllerConfig{
		Scope: scope, Store: store, Publisher: freshPublisher,
		VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil },
		Authorize:   func(context.Context, string) error { return nil },
		Candidates:  func(context.Context) ([]contract.Candidate, error) { return nil, nil },
		Prepare: func(context.Context, []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
			t.Fatal("recovery recompiled user source")
			return nil, nil
		},
		CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
			return nil, nil
		},
		RestoreArtifacts: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
			restored++
			return makeArtifacts(refs), nil
		},
		ValidateArtifact: func(context.Context, *artifactcohort.Artifact) error { return nil },
		NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
			return []authz.Document{doc}, nil
		},
		Fence: func(context.Context, []authz.NamedDependency) error { return nil },
		AdmitAudience: func(ctx context.Context, _ contract.Channel, _ authz.NamedReference) (bool, error) {
			role, _ := ctx.Value(audienceRoleKey{}).(string)
			return role == "operator", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if restored != 1 || prepared != 3 {
		t.Fatalf("restart hydration restored=%d compiled=%d", restored, prepared)
	}
	got, err := restarted.ActiveSnapshot(ctx)
	if err != nil || len(got.Selections) != 1 || got.Selections[0].Bundle.Hash != first.Bundle.Hash {
		t.Fatalf("restart selection: %+v %v", got, err)
	}
}
