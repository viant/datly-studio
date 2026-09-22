package reader

// Output is the generated output scaffold for report.
type Output struct {
	Reports []*Report `parameter:"Reports,kind=output,in=view,dataType=[]*Report" view:"report,type=Report,table=reports,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={namespace,slug,title,status,owner_id,default_connector_name,current_draft_version,updated_at}" sql:"uri=studio_reports_reader_report:sql/report.sql"`
}
