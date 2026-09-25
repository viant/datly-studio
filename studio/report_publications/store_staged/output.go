package store_staged

// Output is the generated output scaffold for publication.
type Output struct {
	Publications []*StagedPublication `parameter:"Publications,kind=output,in=view,dataType=[]*StagedPublication" view:"publication,type=StagedPublication,table=report_publications,selectorNoLimit=true" sql:"uri=studio_report_publications_store_staged_publication:sql/read.sql"`
}
