package sdk

import (
	"context"
	"strings"
)

type namespaceSelectionKey struct{}

// WithNamespaceSelection binds a requested workspace ID. It is selection, not
// authorization: the receiving transport must resolve it against visible rows.
func WithNamespaceSelection(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, namespaceSelectionKey{}, id)
}
func NamespaceSelectionFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(namespaceSelectionKey{}).(string)
	return id, ok
}

// RequiresNamespaceSelection distinguishes resource operations from global
// connector and namespace administration and linked predicate type discovery.
func RequiresNamespaceSelection(operation string) bool {
	return !strings.HasPrefix(operation, "namespaces.") && !strings.HasPrefix(operation, "connectors.") && operation != "authorization_predicates.types"
}
