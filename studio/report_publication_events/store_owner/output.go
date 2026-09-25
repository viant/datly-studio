package store_owner

// Output is the generated output scaffold for report.
type Output struct {
	Reports []*StoredReport `parameter:"Reports,kind=output,in=view,dataType=[]*StoredReport" view:"report,type=StoredReport,table=reports,limit=1" sql:"uri=studio_report_publication_events_store_owner_report:sql/read.sql"`
}
