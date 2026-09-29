// Package versionclone supplies server-only policy continuity for an authorized
// draft clone. It never accepts client-authored policy documents or role facts.
package versionclone

import (
	"context"
	"fmt"
	"strconv"

	"github.com/viant/authz"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
)

type Catalog interface {
	Catalog(context.Context, int, int) ([]*catalog.Entry, error)
}
type PolicyStore interface {
	Get(context.Context, authz.Resource) (authz.Document, error)
	Provision(context.Context, authz.Document, string) (authz.Document, error)
}

// Snapshot reads exact source revisions before any clone writes. Inherited skill
// rows deduplicate to their component policy; explicit skill policies stay exact.
func Snapshot(ctx context.Context, source Catalog, store PolicyStore, componentID, namespaceID string, version int) ([]authz.Document, error) {
	if source == nil || store == nil || componentID == "" || namespaceID == "" || version < 1 {
		return nil, authz.ErrDenied
	}
	documents := []authz.Document{}
	seen := map[authz.Resource]bool{}
	for offset := 0; ; offset += 100 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, err := source.Catalog(ctx, 100, offset)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row == nil {
				return nil, fmt.Errorf("source policy catalog is invalid")
			}
			if row.ComponentID != componentID || row.Version != strconv.Itoa(version) || !row.HasPolicy {
				continue
			}
			if row.NamespaceID != namespaceID {
				return nil, authz.ErrDenied
			}
			resource := authz.Resource{Kind: row.PolicyKind, ID: row.PolicyID, Version: row.Version, Tenant: row.Tenant}
			if resource.Kind != "component" && resource.Kind != "skill" {
				return nil, authz.ErrDenied
			}
			if seen[resource] {
				continue
			}
			seen[resource] = true
			document, err := store.Get(ctx, resource)
			if err != nil {
				return nil, err
			}
			if document.Resource != resource || document.Revision < 1 || len(document.Policies) == 0 {
				return nil, fmt.Errorf("source policy revision is invalid")
			}
			for action, policy := range document.Policies {
				if err := authz.ValidatePolicy(action, policy); err != nil {
					return nil, err
				}
			}
			documents = append(documents, document)
		}
		if len(rows) < 100 {
			return documents, nil
		}
	}
}

// Preserve requires the caller's managed transaction. Provisioning changes only
// version identity; policy administration grants and every restriction are copied.
func Preserve(ctx context.Context, store PolicyStore, documents []authz.Document, version int, actor string) error {
	if store == nil || version < 1 || actor == "" {
		return authz.ErrDenied
	}
	for _, source := range documents {
		target := source
		target.Resource.Version = strconv.Itoa(version)
		target.Revision = 0
		if target.Resource.Version == source.Resource.Version {
			return authz.ErrDenied
		}
		if _, err := store.Provision(ctx, target, actor); err != nil {
			return err
		}
	}
	return nil
}
