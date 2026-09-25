package store_access

// NamespaceAccessScope passes only the server-bound authorization input to
// the typed predicate. This component is not registered on public routes.
func (input Input) NamespaceAccessScope() (subject, name, permission string) {
	return input.Subject, input.Name, input.Permission
}
