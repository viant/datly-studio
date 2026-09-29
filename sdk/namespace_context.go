package sdk

import "context"

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
