package store_preview_definition

// Output is the generated output scaffold for definition.
type Output struct {
	Definitions []*PreviewDefinition `parameter:"Definitions,kind=output,in=view,dataType=[]*PreviewDefinition" view:"definition,type=PreviewDefinition,table=reports,limit=1" sql:"uri=studio_report_versions_store_preview_definition_definition:sql/read.sql"`
}
