package store_catalog

// Output is the generated output scaffold for report.
type Output struct {
	Reports []*StoredReport `parameter:"Reports,kind=output,in=view,dataType=[]*StoredReport" view:"report,type=StoredReport,table=reports,limit=500,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={updated_at,id}" sql:"uri=studio_reports_store_catalog_report:sql/read.sql"`
}
