package artifactrelease_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	release "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

type snapshotCaller struct {
	publisher *artifactcohort.Publisher
	calls     atomic.Int64
}

func (c *snapshotCaller) Discover(context.Context) (artifactcohort.Manifest, error) {
	c.calls.Add(1)
	return c.publisher.Snapshot()
}
func (c *snapshotCaller) Fetch(_ context.Context, in artifactcohort.FetchInput) (*artifactcohort.Artifact, error) {
	c.calls.Add(1)
	return c.publisher.Get(in.Root)
}

type snapshotRoleKey struct{}
type snapshotFacts struct{}

func (snapshotFacts) Resolve(ctx context.Context) (authz.Facts, error) {
	role, _ := ctx.Value(snapshotRoleKey{}).(string)
	return authz.Facts{Subject: role, Tenant: "tenant", Issuer: "verified", Roles: []string{role}, ValidUntil: time.Now().Add(time.Hour)}, nil
}

// The producer owns immutable snapshot persistence. No individual component
// stamp is needed: the controller consumes exact bytes through trusted hooks.
func TestDraftSnapshotHooksPublishRecoverAndRefreshWithoutComponentPublication(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "studio.db"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	scope := sqlstore.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "example"}
	store, err := sqlstore.New(db, scope, "editor")
	require.NoError(t, err)
	persist := func(uri string, version uint64, body string) artifactcohort.RootRef {
		raw := json.RawMessage(body)
		ref := artifactcohort.RootRef{URI: uri, Kind: "window", Version: version, Hash: artifactcohort.Hash(raw)}
		encoded, err := json.Marshal(artifactcohort.Artifact{Root: ref, Payload: raw})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, ref.Hash+".json"), encoded, 0600))
		return ref
	}
	baseline := persist("window://example/baseline", 7, `{"schemaVersion":1,"value":"baseline"}`)
	working := persist("window://example/working", 41, `{"schemaVersion":1,"value":"first draft"}`)
	restore := func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
		out := make([]*artifactcohort.Artifact, 0, len(refs))
		for _, ref := range refs {
			raw, err := os.ReadFile(filepath.Join(dir, ref.Hash+".json"))
			if err != nil {
				return nil, err
			}
			var a artifactcohort.Artifact
			if err = json.Unmarshal(raw, &a); err != nil {
				return nil, err
			}
			if a.Root != ref {
				return nil, artifactcohort.ErrConflict
			}
			out = append(out, &a)
		}
		return out, nil
	}
	resource := authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}
	named := authz.NamedReference{Resource: resource, Revision: 1}
	doc := authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Operators", Policies: map[string]authz.Policy{
		"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
		"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
	}}}}}
	prepares, notifications := 0, 0
	makeController := func(p *artifactcohort.Publisher) *release.Controller {
		_, err := p.ReplaceCatalog(ctx, nil, nil, 0)
		require.NoError(t, err)
		c, err := release.NewController(release.ControllerConfig{
			Scope: scope, Store: store, Publisher: p,
			VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil },
			Authorize:   func(context.Context, string) error { return nil },
			Candidates:  func(context.Context) ([]contract.Candidate, error) { return nil, nil },
			Prepare: func(ctx context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				prepares++
				return restore(ctx, refs)
			},
			RestoreArtifacts: restore,
			AvailableRoot: func(ctx context.Context, ref artifactcohort.RootRef) error {
				_, err := restore(ctx, []artifactcohort.RootRef{ref})
				return err
			},
			CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
				return nil, nil
			},
			ValidateArtifact: func(_ context.Context, a *artifactcohort.Artifact) error { return artifactcohort.ValidateArtifact(a) },
			NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
				return []authz.Document{doc}, nil
			},
			Fence: func(context.Context, []authz.NamedDependency) error { return nil },
			AdmitAudience: func(ctx context.Context, _ contract.Channel, _ authz.NamedReference) (bool, error) {
				return ctx.Value(snapshotRoleKey{}) == "operator", nil
			},
			Notify: func(context.Context) { notifications++ },
		})
		require.NoError(t, err)
		return c
	}
	publisher := artifactcohort.NewPublisher()
	controller := makeController(publisher)
	require.NoError(t, controller.Recover(ctx))
	publish := func(revision uint64, root artifactcohort.RootRef) contract.Result {
		out, err := controller.Publish(ctx, contract.PublishInput{Namespace: "example", Name: "draft", Policies: []authz.NamedReference{named}, Roots: []artifactcohort.RootRef{baseline, root}, ExpectedRevision: revision})
		require.NoError(t, err)
		require.True(t, out.Activated)
		return out
	}
	first := publish(0, working)
	require.Equal(t, "draft", first.Channel.Key.Name)
	// A fresh publisher/controller restores retained bytes, not mutable drafts.
	publisher = artifactcohort.NewPublisher()
	controller = makeController(publisher)
	require.NoError(t, controller.Recover(ctx))
	require.Equal(t, 1, prepares)
	caller := &snapshotCaller{publisher: publisher}
	replica, err := artifactcohort.NewReplica(artifactcohort.Config{Enabled: true, Source: "producer", Tenant: "tenant", Account: "account", Caller: caller, Facts: snapshotFacts{}, SessionKey: func(ctx context.Context) (string, error) {
		role, _ := ctx.Value(snapshotRoleKey{}).(string)
		return role, nil
	}})
	require.NoError(t, err)
	require.NoError(t, replica.Sync(ctx))
	allowed := context.WithValue(ctx, snapshotRoleKey{}, "operator")
	denied := context.WithValue(ctx, snapshotRoleKey{}, "viewer")
	require.NoError(t, replica.Admit(allowed))
	require.ErrorIs(t, replica.Admit(denied), artifactcohort.ErrDenied)
	before := caller.calls.Load()
	for i := 0; i < 25; i++ {
		view, err := replica.Open(allowed, working.URI)
		require.NoError(t, err)
		require.Equal(t, working, view.Ref())
	}
	require.Equal(t, before, caller.calls.Load(), "interactive opens must not fetch producer metadata")
	updated := persist(working.URI, 42, `{"schemaVersion":1,"value":"updated draft"}`)
	second := publish(first.Channel.Revision, updated)
	require.NotEqual(t, first.Bundle.Hash, second.Bundle.Hash)
	require.GreaterOrEqual(t, notifications, 2)
	require.NoError(t, replica.Sync(ctx))
	require.NoError(t, replica.EnsureAdmitted(allowed))
	view, err := replica.Open(allowed, updated.URI)
	require.NoError(t, err)
	require.Equal(t, updated, view.Ref())
	retained, err := replica.Open(allowed, baseline.URI)
	require.NoError(t, err)
	require.Equal(t, baseline, retained.Ref())
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions`).Scan(&count))
	require.Zero(t, count, "snapshot release must not create individual component versions")
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_publications`).Scan(&count))
	require.Zero(t, count)
}
