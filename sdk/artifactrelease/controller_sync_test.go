package artifactrelease_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/releasegate"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

func TestTwoControllersSynchronizeCommittedReleaseAndWithdrawals(t *testing.T) {
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
	policy := authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}
	ref := authz.NamedReference{Resource: policy, Revision: 1}
	var head atomic.Int64
	head.Store(1)
	var available atomic.Bool
	available.Store(true)
	var infrastructureDown atomic.Bool
	sourceUnavailable := errors.New("source database unavailable")
	var blockSource atomic.Bool
	blockedSourceEntered := make(chan struct{})
	var policyAvailable atomic.Bool
	policyAvailable.Store(true)
	root := func(version uint64) artifactcohort.RootRef {
		body, _ := json.Marshal(map[string]uint64{"version": version})
		return artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: version, Hash: artifactcohort.Hash(body)}
	}
	artifact := func(ref artifactcohort.RootRef) *artifactcohort.Artifact {
		body, _ := json.Marshal(map[string]uint64{"version": ref.Version})
		return &artifactcohort.Artifact{Root: ref, Payload: body}
	}
	var restoredB, notifiedB atomic.Int64
	var raceArmed atomic.Bool
	raceReached := make(chan struct{})
	raceResume := make(chan struct{})
	makeController := func(second bool) (*sdkrelease.Controller, *artifactcohort.Publisher) {
		store, e := sqlstore.New(db, scope, "verified-editor")
		if e != nil {
			t.Fatal(e)
		}
		publisher := artifactcohort.NewPublisher()
		if _, e = publisher.ReplaceCatalog(ctx, nil, nil, 0); e != nil {
			t.Fatal(e)
		}
		controller, e := sdkrelease.NewController(sdkrelease.ControllerConfig{
			Scope: scope, Store: store, Publisher: publisher,
			VerifyScope: func(context.Context) (sqlstore.Scope, error) { return scope, nil },
			Authorize:   func(context.Context, string) error { return nil },
			Candidates: func(context.Context) ([]contract.Candidate, error) {
				return []contract.Candidate{{Root: root(1)}, {Root: root(2)}}, nil
			},
			Prepare: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				out := make([]*artifactcohort.Artifact, 0, len(refs))
				for _, r := range refs {
					out = append(out, artifact(r))
				}
				return out, nil
			},
			CompileBindings: func(context.Context, contract.Bundle, []*artifactcohort.Artifact) ([]sqlstore.ComponentBinding, error) {
				return nil, nil
			},
			RestoreArtifacts: func(_ context.Context, refs []artifactcohort.RootRef) ([]*artifactcohort.Artifact, error) {
				if second {
					restoredB.Add(1)
				}
				out := make([]*artifactcohort.Artifact, 0, len(refs))
				for _, r := range refs {
					out = append(out, artifact(r))
				}
				return out, nil
			},
			AvailableRoot: func(callCtx context.Context, _ artifactcohort.RootRef) error {
				if second && blockSource.CompareAndSwap(true, false) {
					close(blockedSourceEntered)
					<-callCtx.Done()
					return callCtx.Err()
				}
				if infrastructureDown.Load() {
					return sourceUnavailable
				}
				if !available.Load() {
					return contract.ErrInvalid
				}
				return nil
			},
			ValidateArtifact: func(context.Context, *artifactcohort.Artifact) error { return nil },
			NamedDocuments: func(context.Context, []authz.NamedReference) ([]authz.Document, error) {
				if !policyAvailable.Load() {
					return nil, authz.ErrDenied
				}
				revision := head.Load()
				return []authz.Document{{Resource: policy, Revision: revision, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Operators", Policies: map[string]authz.Policy{
					"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
					"execute":  {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "operator"}},
				}}}}}}, nil
			},
			Fence: func(_ context.Context, deps []authz.NamedDependency) error {
				if second && raceArmed.CompareAndSwap(true, false) {
					close(raceReached)
					<-raceResume
				}
				if !policyAvailable.Load() {
					return authz.ErrDenied
				}
				for _, d := range deps {
					if d.Revision != head.Load() {
						return authz.ErrDenied
					}
				}
				return nil
			},
			AdmitAudience: func(context.Context, contract.Channel, authz.NamedReference) (bool, error) { return true, nil },
			Notify: func(context.Context) {
				if second {
					notifiedB.Add(1)
				}
			},
		})
		if e != nil {
			t.Fatal(e)
		}
		if e = controller.Recover(ctx); e != nil {
			t.Fatal(e)
		}
		return controller, publisher
	}
	a, _ := makeController(false)
	b, publishedB := makeController(true)
	assertSelected := func(want uint64) {
		t.Helper()
		manifest, e := publishedB.Snapshot()
		if e != nil {
			t.Fatal(e)
		}
		if want == 0 {
			if len(manifest.Cohorts) != 0 {
				t.Fatalf("stale cohorts: %+v", manifest.Cohorts)
			}
			return
		}
		if len(manifest.Cohorts) != 1 || len(manifest.Cohorts[0].Roots) != 1 || manifest.Cohorts[0].Roots[0] != root(want) {
			t.Fatalf("selected=%+v,want version %d", manifest.Cohorts, want)
		}
	}
	first, e := a.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: "main", Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{root(1)}})
	if e != nil {
		t.Fatal(e)
	}
	if changed, e := b.SyncOnce(ctx); e != nil || !changed {
		t.Fatalf("first sync changed=%v err=%v", changed, e)
	}
	assertSelected(1)
	reads, notifications := restoredB.Load(), notifiedB.Load()
	if changed, e := b.SyncOnce(ctx); e != nil || changed || restoredB.Load() != reads || notifiedB.Load() != notifications {
		t.Fatalf("unchanged sync rebuilt or notified: %v %v", changed, e)
	}
	type syncResult struct {
		changed bool
		err     error
	}
	raced := make(chan syncResult, 1)
	raceArmed.Store(true)
	go func() { changed, err := b.SyncOnce(ctx); raced <- syncResult{changed, err} }()
	<-raceReached // B has read the old committed channel but has not finished its fingerprint.
	second, e := a.Publish(ctx, contract.PublishInput{Namespace: "demo", Name: "main", Policies: []authz.NamedReference{ref}, Roots: []artifactcohort.RootRef{root(2)}, ExpectedRevision: first.Channel.Revision})
	if e != nil {
		t.Fatal(e)
	}
	close(raceResume)
	if result := <-raced; result.err != nil || result.changed {
		t.Fatalf("raced old fingerprint unexpectedly projected: %+v", result)
	}
	assertSelected(1)
	if changed, e := b.SyncOnce(ctx); e != nil || !changed {
		t.Fatalf("revision sync changed=%v err=%v", changed, e)
	}
	assertSelected(2)
	available.Store(false)
	if changed, e := b.SyncOnce(ctx); e != nil || !changed {
		t.Fatalf("withdrawal sync changed=%v err=%v", changed, e)
	}
	assertSelected(0)
	available.Store(true)
	if _, e = a.Promote(ctx, contract.PromoteInput{Namespace: "demo", Name: "main", BundleHash: first.Bundle.Hash, ExpectedRevision: second.Channel.Revision}); e != nil {
		t.Fatal(e)
	}
	if changed, e := b.SyncOnce(ctx); e != nil || !changed {
		t.Fatalf("rollback sync changed=%v err=%v", changed, e)
	}
	assertSelected(1)
	head.Store(2)
	if err = a.RefreshPolicies(ctx, policy); err != nil {
		t.Fatal(err)
	}
	if changed, e := b.SyncOnce(ctx); e != nil || !changed {
		t.Fatalf("policy update changed=%v err=%v", changed, e)
	}
	assertSelected(1)
	policyAvailable.Store(false)
	if err = a.RefreshPolicies(ctx, policy); err == nil {
		t.Fatal("archived policy remained active")
	}
	if changed, e := b.SyncOnce(ctx); e != nil || !changed {
		t.Fatalf("policy withdrawal changed=%v err=%v", changed, e)
	}
	assertSelected(0)
	if notifications != 1 || notifiedB.Load() != 6 {
		t.Fatalf("notifications=%d,want6", notifiedB.Load())
	}
	// An in-flight Stage changes only pending fields; B must not project it or
	// repeatedly rebuild the last committed channel.
	stageStore, e := sqlstore.New(db, scope, "verified-editor")
	if e != nil {
		t.Fatal(e)
	}
	current, e := stageStore.Channel(ctx, first.Channel.Key)
	if e != nil {
		t.Fatal(e)
	}
	pending, e := contract.PreparePromotion(current, current.Revision, *first.Bundle)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = stageStore.StageRevision(ctx, current, pending, *first.Bundle); e != nil {
		t.Fatal(e)
	}
	reads, notifications = restoredB.Load(), notifiedB.Load()
	if changed, e := b.SyncOnce(ctx); e != nil || changed || restoredB.Load() != reads || notifiedB.Load() != notifications {
		t.Fatalf("pending-only change rebuilt or notified: changed=%v err=%v", changed, e)
	}
	infrastructureDown.Store(true)
	if _, e := b.SyncOnce(ctx); !errors.Is(e, sourceUnavailable) {
		t.Fatalf("source infrastructure error treated as withdrawal: %v", e)
	}
	if e := b.Gate().Read(func() error { return nil }); !errors.Is(e, releasegate.ErrNotReady) {
		t.Fatalf("source error left release serving ready: %v", e)
	}
	infrastructureDown.Store(false)
	beforeRecovery, _ := publishedB.Snapshot()
	if changed, e := b.SyncOnce(ctx); e != nil || changed {
		afterRecovery, _ := publishedB.Snapshot()
		t.Fatalf("source recovery did not restore unchanged committed state: %v %v before=%+v after=%+v", changed, e, beforeRecovery, afterRecovery)
	}
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan error, 1)
	blockSource.Store(true)
	go func() { workerDone <- b.RunSync(workerCtx, time.Millisecond, nil) }()
	select {
	case <-blockedSourceEntered:
	case <-time.After(time.Second):
		t.Fatal("background source probe did not start")
	}
	stopWorker()
	select {
	case e := <-workerDone:
		if e != nil {
			t.Fatalf("canceled background worker: %v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not join after cancellation during source read")
	}
	if changed, e := b.SyncOnce(ctx); e != nil || changed {
		t.Fatalf("worker cancellation did not restore unchanged committed state: %v %v", changed, e)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = b.SyncOnce(ctx); err == nil {
		t.Fatal("closed store was treated as an unchanged healthy snapshot")
	}
	if err = b.Gate().Read(func() error { return nil }); !errors.Is(err, releasegate.ErrNotReady) {
		t.Fatalf("release reads remained ready after sync failure: %v", err)
	}
	workerCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err = b.RunSync(workerCtx, time.Millisecond, nil); err != nil {
		t.Fatalf("background retry did not stop on cancellation: %v", err)
	}
}
