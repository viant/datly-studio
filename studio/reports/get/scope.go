package get

// ReportCatalogScope supplies the typed predicate with the subject from the
// required auth-context component. No request field can override this scope.
func (input ReportGetInput) ReportCatalogScope() (string, bool) {
	if input.Auth == nil || input.Auth.Auth == nil {
		return "", true
	}
	return input.Auth.Auth.Subject, true
}
