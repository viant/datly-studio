package store_mcp_names

// MCPNameReportID exposes the server-bound report identity to the typed
// predicate. This component is not registered on public routes.
func (input Input) MCPNameReportID() string { return input.ReportId }
