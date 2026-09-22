package reader

// Output is the generated output scaffold for publication.
type Output struct {
	Publications []*ReportPublication `parameter:"Publications,kind=output,in=view,dataType=[]*ReportPublication" view:"publication,type=ReportPublication,table=report_publications" sql:"uri=studio_report_publications_reader_publication:sql/read.sql"`
}
