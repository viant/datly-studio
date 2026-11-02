package namespacemcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type controlledStaleDefinitions struct {
	snapshot []Definition
	captured chan struct{}
	release  chan struct{}
}

func (d *controlledStaleDefinitions) Load(ctx context.Context) ([]Definition, error) {
	snapshot := append([]Definition(nil), d.snapshot...)
	close(d.captured)
	select {
	case <-d.release:
		return snapshot, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// A successful synchronous publication must survive reconciliation of an older
// persisted snapshot, including a commit that retains the same endpoint.
// No server, database, network request, or sleep is involved.
func TestStaleSnapshotMustNotRemoveSuccessfulPublication(t *testing.T) {
	for _, scenario := range []string{"absent", "disabled", "unchanged_successful_commit"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			manager, err := NewHTTP(func(context.Context, Endpoint) (http.Handler, error) {
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			// Install shared-listener state without opening a socket. Requests exercise
			// the production shared router directly through httptest.ResponseRecorder.
			manager.sharedServer = &http.Server{}
			manager.sharedAddress = "127.0.0.1:12345"
			manager.sharedPort = 12345
			manager.sharedActive = true
			defer manager.Close(ctx)
			id := strings.Repeat("a", 64)
			routeStatus := func() int {
				recorder := httptest.NewRecorder()
				manager.serveShared(recorder, httptest.NewRequest(http.MethodPost, "/namespaces/"+id+"/mcp", nil))
				return recorder.Code
			}
			definitions := &controlledStaleDefinitions{captured: make(chan struct{}), release: make(chan struct{})}
			if scenario != "absent" {
				definitions.snapshot = []Definition{{NamespaceID: id, Enabled: false, Active: true, Port: 0}}
			}
			var before *instance
			if scenario == "unchanged_successful_commit" {
				if _, err = manager.Start(ctx, id, 0); err != nil {
					t.Fatal(err)
				}
				before = manager.entries[id]
				if got := routeStatus(); got != 200 {
					t.Fatalf("route before Apply = %d, want 200", got)
				}
			}
			reconciler := &Reconciler{Manager: manager, Definitions: definitions}
			done := make(chan error, 1)
			go func() { _, e := reconciler.Apply(ctx); done <- e }()
			select {
			case <-definitions.captured:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			committed := false
			if _, err = manager.RebindWithCommit(ctx, id, 0, func(Endpoint) error { committed = true; return nil }); err != nil {
				t.Fatal(err)
			}
			if !committed {
				t.Fatal("publication commit callback did not execute")
			}
			if before != nil && before != manager.entries[id] {
				t.Fatal("unchanged commit unexpectedly replaced entry")
			}
			if got := routeStatus(); got != 200 {
				t.Fatalf("route after successful publication, before stale snapshot apply = %d, want 200", got)
			}
			close(definitions.release)
			select {
			case err = <-done:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if err != nil {
				t.Fatal(err)
			}
			got := routeStatus()
			t.Logf("successful publication route before stale application=200; after=%d", got)
			if got != 200 {
				t.Errorf("stale %s snapshot removed successful publication: route=%d, want 200", scenario, got)
			}
		})
	}
}

func snapshotManager(ctx context.Context, t *testing.T) *Manager {
	t.Helper()
	manager, err := NewHTTP(func(context.Context, Endpoint) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer permitted" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	manager.sharedServer = &http.Server{}
	manager.sharedAddress = "127.0.0.1:12345"
	manager.sharedPort = 12345
	manager.sharedActive = true
	return manager
}

func snapshotRouteStatus(ctx context.Context, manager *Manager, id string, authorized bool) int {
	request := httptest.NewRequest(http.MethodPost, "/namespaces/"+id+"/mcp", nil).WithContext(ctx)
	if authorized {
		request.Header.Set("Authorization", "Bearer permitted")
	}
	response := httptest.NewRecorder()
	manager.serveShared(response, request)
	return response.Code
}

func blockedSnapshot(ctx context.Context, t *testing.T, manager *Manager, snapshot []Definition) func() {
	t.Helper()
	definitions := &controlledStaleDefinitions{snapshot: snapshot, captured: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, err := (&Reconciler{Manager: manager, Definitions: definitions}).Apply(ctx); done <- err }()
	select {
	case <-definitions.captured:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	return func() {
		t.Helper()
		close(definitions.release)
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

func TestStaleSnapshotCannotStartOrRebindAfterCommittedControl(t *testing.T) {
	for _, scenario := range []string{"start_after_withdrawal", "rebind_after_unchanged_commit"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			manager := snapshotManager(ctx, t)
			defer manager.Close(ctx)
			id := strings.Repeat("b", 64)
			desired := Definition{NamespaceID: id, Active: true, Enabled: true}
			if scenario == "rebind_after_unchanged_commit" {
				if _, err := manager.Start(ctx, id, 0); err != nil {
					t.Fatal(err)
				}
				occupied, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer occupied.Close()
				desired.Port = occupied.Addr().(*net.TCPAddr).Port // A stale rebind must not even attempt reservation.
			}
			finish := blockedSnapshot(ctx, t, manager, []Definition{desired})
			if scenario == "start_after_withdrawal" {
				if err := manager.StopWithCommit(ctx, id, func(Endpoint) error { return nil }); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := manager.RebindWithCommit(ctx, id, 0, func(Endpoint) error { return nil }); err != nil {
					t.Fatal(err)
				}
			}
			finish()
			if scenario == "start_after_withdrawal" {
				if got := snapshotRouteStatus(ctx, manager, id, true); got != 404 {
					t.Fatalf("stale snapshot restarted withdrawn route: %d", got)
				}
			} else {
				endpoint, exists := manager.Get(id)
				if !exists || endpoint.Port != 12345 {
					t.Fatalf("stale snapshot rebound committed endpoint: %+v, present=%v", endpoint, exists)
				}
				if got := snapshotRouteStatus(ctx, manager, id, false); got != 403 {
					t.Fatalf("handler authorization changed: %d", got)
				}
				if got := snapshotRouteStatus(ctx, manager, id, true); got != 200 {
					t.Fatalf("committed shared route unavailable: %d", got)
				}
			}
		})
	}
}

func TestFailedCommitDoesNotSupersedePersistedSnapshot(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			manager := snapshotManager(ctx, t)
			defer manager.Close(ctx)
			id := strings.Repeat("c", 64)
			if existing {
				if _, err := manager.Start(ctx, id, 0); err != nil {
					t.Fatal(err)
				}
			}
			finish := blockedSnapshot(ctx, t, manager, []Definition{{NamespaceID: id, Active: true, Enabled: !existing}})
			failure := errors.New("persistence rejected")
			if _, err := manager.RebindWithCommit(ctx, id, 0, func(Endpoint) error { return failure }); !errors.Is(err, failure) {
				t.Fatalf("commit failure lost: %v", err)
			}
			finish()
			want := 200
			if existing {
				want = 404
			}
			if got := snapshotRouteStatus(ctx, manager, id, true); got != want {
				t.Fatalf("failed commit superseded persisted state: route=%d want=%d", got, want)
			}
		})
	}
}

func TestSupersededNamespaceDoesNotBlockIndependentConvergenceOrFreshWithdrawal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	manager := snapshotManager(ctx, t)
	defer manager.Close(ctx)
	first, second := strings.Repeat("d", 64), strings.Repeat("e", 64)
	snapshot := []Definition{{NamespaceID: second, Active: true, Enabled: true}}
	finish := blockedSnapshot(ctx, t, manager, snapshot)
	if _, err := manager.RebindWithCommit(ctx, first, 0, func(Endpoint) error { return nil }); err != nil {
		t.Fatal(err)
	}
	finish()
	for _, id := range []string{first, second} {
		if got := snapshotRouteStatus(ctx, manager, id, true); got != 200 {
			t.Fatalf("independent route %s unavailable: %d", id, got)
		}
	}
	fresh := &controlledStaleDefinitions{snapshot: snapshot, captured: make(chan struct{}), release: make(chan struct{})}
	close(fresh.release)
	if _, err := (&Reconciler{Manager: manager, Definitions: fresh}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := snapshotRouteStatus(ctx, manager, first, true); got != 404 {
		t.Fatalf("fresh withdrawal ignored: %d", got)
	}
	if got := snapshotRouteStatus(ctx, manager, second, true); got != 200 {
		t.Fatalf("unrelated namespace changed: %d", got)
	}
}
