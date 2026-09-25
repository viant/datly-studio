package store_global_access

// Output is the generated output scaffold for report.
type Output struct {
	Reports []*StoredReport `parameter:"Reports,kind=output,in=view,dataType=[]*StoredReport" view:"report,type=StoredReport,table=reports,limit=1" sql:"uri=studio_reports_store_global_access_report:sql/read.sql"`
}
