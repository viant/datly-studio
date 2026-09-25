package store_active_others

// ExcludedPublicationReport passes the server-bound current report ID to the
// typed predicate; this component is never publicly routed.
func (input Input) ExcludedPublicationReport() string { return input.ExcludeReportId }
