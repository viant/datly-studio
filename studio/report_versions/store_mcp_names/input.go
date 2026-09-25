package store_mcp_names

// Input is the generated input scaffold for version.
type Input struct {
	ReportId string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/report_versions/mcpnamepredicate.OtherReportCandidate"`
	Has      *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId bool
}
