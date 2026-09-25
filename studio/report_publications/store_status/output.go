package store_status

// Output is the generated output scaffold for publication.
type Output struct {
	Publications []*StoredPublication `parameter:"Publications,kind=output,in=view,dataType=[]*StoredPublication" view:"publication,type=StoredPublication,table=report_publications,limit=1" sql:"uri=studio_report_publications_store_status_publication:sql/read.sql"`
}
