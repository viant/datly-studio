package host

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/runtimevalidation"
	"github.com/viant/datly-studio/studio/namespaces/change"
	"github.com/viant/datly-studio/studio/settings"
	xresponse "github.com/viant/xdatly/response"
)

// NamespaceSettingsCommitted refreshes policy before attempting a rebuild.
// Failed builds retain old internal capabilities, but cannot restore public
// access forbidden by the committed policy. Other namespaces are untouched.
func (g *NamespaceGroup) NamespaceSettingsCommitted(ctx context.Context, identities []change.Identity) error {
	if g == nil || g.runtimePolicies == nil || g.Listeners == nil {
		return fmt.Errorf("namespace runtime is unavailable")
	}
	var result error
	seen := map[string]bool{}
	var reload []string
	for _, identity := range identities {
		if !identity.SettingsChanged {
			continue
		}
		id := namespaceaccess.ID(identity.OwnerID, identity.Name)
		if identity.OwnerID == "" || identity.Name == "" || identity.Revision <= 0 || id != identity.NamespaceID {
			result = errors.Join(result, fmt.Errorf("invalid committed namespace identity"))
			continue
		}
		if identity.SecurityChanged {
			g.runtimePolicies.Update(id, identity.Revision, &settings.Namespace{Security: identity.Security})
		}
		if !seen[id] {
			reload = append(reload, id)
			seen[id] = true
		}
	}
	// Publish all committed restrictions before waiting for any compilation.
	controlContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	for _, id := range reload {
		var err error
		if _, running := g.Listeners.Get(id); running {
			err = g.Listeners.Reload(controlContext, id, 0)
		}
		g.failureMu.Lock()
		if g.failures == nil {
			g.failures = map[string]error{}
		}
		if err == nil {
			delete(g.failures, id)
		} else {
			g.failures[id] = err
		}
		g.failureMu.Unlock()
		if err != nil {
			result = errors.Join(result, fmt.Errorf("namespace settings committed but runtime refresh failed: %w", err))
		}
	}
	if result != nil {
		return &xresponse.Error{Code: http.StatusConflict, Cause: result, Payload: map[string]any{
			"status": "error", "code": "namespace_settings_refresh_failed", "committed": true,
			"message":     "Settings were saved, but published components could not be reloaded. The committed security rules are enforced.",
			"diagnostics": runtimevalidation.Diagnostics(result),
		}}
	}
	return nil
}
