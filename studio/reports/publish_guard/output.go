package publish_guard

// Output is the generated output scaffold for report.
type Output struct {
	Item *Report `parameter:"Item,kind=output,in=view,dataType=*Report" json:"-" view:"report,type=Report,table=components,limit=1" sql:"uri=studio_reports_publish_guard_report:sql/report.sql"`
}
