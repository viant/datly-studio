package store_mcp_names

// Output is the generated output scaffold for version.
type Output struct {
	Versions []*NameSource `parameter:"Versions,kind=output,in=view,dataType=[]*NameSource" view:"version,type=NameSource,table=reports,selectorNoLimit=true" sql:"uri=studio_report_versions_store_mcp_names_version:sql/read.sql"`
}
