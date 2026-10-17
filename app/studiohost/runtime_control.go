package studiohost

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	runtimehost "github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly-studio/sdk"
)

// Process-owned callbacks share the namespace manager with authoring. Nothing
// listens for administration commands and no administration credential exists.
func inProcessControl(group *atomic.Pointer[runtimehost.NamespaceGroup]) *runtimeadmin.Client {
	selected := func(ctx context.Context) (*runtimehost.NamespaceGroup, string, error) {
		id, ok := sdk.NamespaceSelectionFromContext(ctx)
		decoded, err := hex.DecodeString(id)
		current := group.Load()
		if current == nil || !ok || err != nil || len(decoded) != 32 || id != strings.ToLower(id) {
			return nil, "", errors.New("canonical namespace selection is required")
		}
		return current, id, nil
	}
	return runtimeadmin.NewInProcess(func(ctx context.Context, generation int64) error {
		current, id, err := selected(ctx)
		if err != nil {
			return err
		}
		return current.Listeners.Reload(ctx, id, generation)
	}, func(ctx context.Context) (*sdk.RuntimeHost, error) {
		current, id, err := selected(ctx)
		if err != nil {
			return nil, err
		}
		endpoint, mode, revision, ready := current.Listeners.RuntimeStatus(id)
		if !ready {
			return nil, runtimeadmin.ErrNamespaceUnavailable
		}
		return &sdk.RuntimeHost{NamespaceID: id, MCPURL: endpoint.URL(), AuthenticationMode: mode, Status: "ready", Revision: revision, CheckedAt: time.Now().UTC()}, nil
	}, func(ctx context.Context) (json.RawMessage, error) {
		current, id, err := selected(ctx)
		if err != nil {
			return nil, err
		}
		return current.Listeners.Connection(ctx, id)
	})
}
