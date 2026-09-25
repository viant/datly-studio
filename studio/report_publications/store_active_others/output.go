package store_active_others

// Output is the generated output scaffold for publication.
type Output struct {
	Publications []*ActivePublication `parameter:"Publications,kind=output,in=view,dataType=[]*ActivePublication" view:"publication,type=ActivePublication,table=report_publications,selectorNoLimit=true" sql:"uri=studio_report_publications_store_active_others_publication:sql/read.sql"`
}
