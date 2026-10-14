package sdk

import "github.com/viant/datly-studio/internal/namespaceaccess"

// NamespaceID returns the shared storage identity for an immutable owner/name
// namespace key. Embedding hosts that create namespaces must use this contract
// so components, policy catalogs and namespace listeners share the same boundary.
// This is an identity constructor, not an authorization decision.
func NamespaceID(owner, name string) string { return namespaceaccess.ID(owner, name) }
