package store_global_access

// GlobalPublishSubject exposes only the server-bound verified principal to
// the typed predicate. This component is not registered on public routes.
func (input Input) GlobalPublishSubject() string { return input.Subject }
