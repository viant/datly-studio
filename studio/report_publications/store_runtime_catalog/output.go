package store_runtime_catalog

// Output is the generated output scaffold for publication.
type Output struct {
	Publications []*PublishedComponent `parameter:"Publications,kind=output,in=view,dataType=[]*PublishedComponent" view:"publication,type=PublishedComponent,table=report_publications,selectorNoLimit=true" sql:"uri=studio_report_publications_store_runtime_catalog_publication:sql/read.sql"`
}
