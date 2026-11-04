package host

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/scy/auth/jwt"
	xresponse "github.com/viant/xdatly/response"
)

// authorizeNamespace is independent of resource ACL and is re-evaluated for
// discovery and execution. Namespace selection is deployment-owned here.
func (s *Service) authorizeNamespace(ctx context.Context) error {
	if s.config.NamespaceID == "" {
		return nil
	}
	var row namespacemcp.Definition
	var policy namespaceaccess.Policy
	var err error
	if s.config.namespaceDefinitions != nil {
		var found bool
		row, policy, found = s.config.namespaceDefinitions.LookupAccess(s.config.NamespaceID)
		if !found {
			return namespaceDenied()
		}
	} else {
		// Standalone runtimes retain their existing storage-backed lifecycle.
		rows, readErr := (namespacemcp.SQLDefinitions{DB: s.studio}).LoadNamespace(ctx, s.config.NamespaceID)
		if readErr != nil || len(rows) != 1 {
			return namespaceDenied()
		}
		row = rows[0]
		policy = namespaceaccess.Policy{ID: row.NamespaceID, OwnerID: row.OwnerID, Visibility: row.Visibility, Roles: row.AllowedRoles}
	}
	if row.NamespaceID != s.config.NamespaceID || !row.Active {
		return namespaceDenied()
	}
	if policy.CanView(authz.Facts{}, time.Now()) {
		return nil
	}
	var facts authz.Facts
	if s.resourceAccess != nil && s.resourceAccess.Provider != nil {
		facts, err = s.resourceAccess.Provider.Resolve(ctx)
		if err != nil {
			return namespaceDenied()
		}
	} else {
		claims, _ := ctx.Value(verifiedClaimsKey{}).(*jwt.Claims)
		if claims == nil || claims.ExpiresAt == nil {
			return namespaceDenied()
		}
		issuer := claims.Issuer
		if issuer == "" {
			issuer = "datly:verified-default"
		}
		facts = authz.Facts{Subject: claims.Subject, Issuer: issuer, ValidUntil: claims.ExpiresAt.Time}
	}
	if !policy.CanView(facts, time.Now()) {
		return namespaceDenied()
	}
	return nil
}
func namespaceDenied() error {
	return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("namespace access is required")}
}
