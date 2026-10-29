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
	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/datly-studio/sdk"
	dexec "github.com/viant/datly/exec"
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
	return runtimeadmin.NewInProcessWithCache(func(ctx context.Context, generation int64) error {
		current, id, err := selected(ctx)
		if err != nil {
			return err
		}
		// Authoring can enable a namespace and publish before the next periodic
		// reconciliation. Establish its persisted listener synchronously and
		// return an actual build error instead of a misleading missing runtime.
		definitions, err := (namespacemcp.SQLDefinitions{DB: current.DB}).LoadNamespace(ctx, id)
		if err != nil {
			return err
		}
		if len(definitions) != 1 || !definitions[0].Active || !definitions[0].Enabled {
			return runtimeadmin.ErrNamespaceUnavailable
		}
		_, err = current.Listeners.RebindAndReload(ctx, id, definitions[0].Port, generation)
		return err
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
	}, func(ctx context.Context, reportID string, version int, routePath, view string) ([]dexec.CacheInvalidation, error) {
		current, id, err := selected(ctx)
		if err != nil {
			return nil, err
		}
		return current.Listeners.InvalidatePublishedCacheView(ctx, id, reportID, version, routePath, view)
	})
}
