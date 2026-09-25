package store_catalog

// ReportCatalogScope exposes the explicitly bound, server-owned SDK scope to
// the typed Datly predicate. The caller must derive these fields from a
// verified principal; this component is never registered as a public route.
func (input Input) ReportCatalogScope() (subject string, scoped bool) {
	return input.Subject, input.Scoped
}
