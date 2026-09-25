package reader

// ReportCatalogScope supplies the typed predicate with a principal obtained
// through the required auth-context component, never from a caller parameter.
// An unavailable context deliberately returns an empty scoped subject so the
// predicate fails closed.
func (input Input) ReportCatalogScope() (subject string, scoped bool) {
	if input.Auth == nil || input.Auth.Auth == nil {
		return "", true
	}
	return input.Auth.Auth.Subject, true
}
