package host

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/namespacemcp"
)

type failureDefinitions struct {
	values []namespacemcp.Definition
	err    error
}

func (d *failureDefinitions) Load(context.Context) ([]namespacemcp.Definition, error) {
	return d.values, d.err
}

func TestNamespaceReconciliationRetainsFailureUntilRecovery(t *testing.T) {
	ctx := context.Background()
	failedID, healthyID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	cause := errors.New("synthetic reader initialization failure")
	failing := true
	manager, err := namespacemcp.NewHTTP(func(_ context.Context, endpoint namespacemcp.Endpoint) (http.Handler, error) {
		if endpoint.NamespaceID == failedID && failing {
			return nil, cause
		}
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(ctx)
	definitions := &failureDefinitions{values: []namespacemcp.Definition{{NamespaceID: failedID, Active: true, Enabled: true}, {NamespaceID: healthyID, Active: true, Enabled: true}}}
	group := &NamespaceGroup{Listeners: manager, reconciler: &namespacemcp.Reconciler{Manager: manager, Definitions: definitions}}
	if _, err = group.Reconcile(ctx); !errors.Is(err, cause) {
		t.Fatalf("expected original cause, got %v", err)
	}
	if !errors.Is(group.RuntimeFailure(failedID), cause) {
		t.Fatal("failed namespace lost its cause")
	}
	if _, ok := manager.Get(healthyID); !ok || group.RuntimeFailure(healthyID) != nil {
		t.Fatal("independent healthy namespace was affected")
	}
	definitions.err = errors.New("temporary catalog read failure")
	_, _ = group.Reconcile(ctx)
	if !errors.Is(group.RuntimeFailure(failedID), cause) {
		t.Fatal("catalog outage erased the known failure")
	}
	definitions.err = nil
	failing = false
	if _, err = group.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if group.RuntimeFailure(failedID) != nil {
		t.Fatal("successful recovery retained a stale failure")
	}
}
