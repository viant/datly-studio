package sdk

import (
	"context"

	"github.com/viant/datly-studio/internal/namespaceroles"
	"github.com/viant/datly-studio/internal/namespacevisibility"
)

// CanManageNamespace evaluates a stored namespace using the same verified
// provider and delegation rules as the native SDK. Embedding hosts must load
// this record from authoritative storage, never from request parameters.
func CanManageNamespace(ctx context.Context, subject string, namespace Namespace) bool {
	if namespace.Status != "active" || namespace.NamespaceID == "" {
		return false
	}
	encoded, err := (namespaceroles.Grants{View: namespace.AllowedRoles, Manage: namespace.ManagerRoles}).Encode()
	return err == nil && namespacevisibility.CanManage(ctx, subject, namespace.OwnerID, &encoded)
}

// CanViewNamespace evaluates the same persisted namespace audience as native
// SDK queries. Embedding hosts supply a server-loaded record and verified user.
func CanViewNamespace(ctx context.Context, subject string, namespace Namespace) bool {
	if namespace.Status != "active" || namespace.NamespaceID == "" {
		return false
	}
	encoded, err := (namespaceroles.Grants{View: namespace.AllowedRoles, Manage: namespace.ManagerRoles}).Encode()
	return err == nil && namespacevisibility.CanView(ctx, subject, namespace.OwnerID, namespace.Visibility, &encoded)
}

// DecodeNamespaceRoles reads the native backward-compatible grant document.
// Consumers use it only on a persisted namespace record.
func DecodeNamespaceRoles(encoded *string) (view, manage []string, err error) {
	grants, err := namespaceroles.Decode(encoded)
	return grants.View, grants.Manage, err
}
