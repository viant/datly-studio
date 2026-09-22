package reader

// Output is the generated output scaffold for view.
type Output struct {
	Views []*ReportView `parameter:"Views,kind=output,in=view,dataType=[]*ReportView" view:"view,type=ReportView,table=report_views" sql:"uri=studio_report_views_reader_view:sql/view.sql"`
}
