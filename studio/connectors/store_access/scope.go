package store_access

// ConnectorAccessScope passes server-bound authorization input to the typed
// predicate. This component is not registered on public routes.
func (input Input) ConnectorAccessScope() (subject, name, permission string) {
	return input.Subject, input.Name, input.Permission
}
