package reader

// Output is the generated output scaffold for version.
type Output struct {
	Versions []*ReportVersion `parameter:"Versions,kind=output,in=view,dataType=[]*ReportVersion" view:"version,type=ReportVersion,table=report_versions,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={version_no,state,authoring_mode,compile_status,source_revision,created_by,created_at,validated_at,published_at}" sql:"uri=studio_report_versions_reader_version:sql/version.sql"`
}
