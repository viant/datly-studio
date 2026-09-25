package reader

// Output is the generated output scaffold for report.
type Output struct {
	Items      []*Report `parameter:"Items,kind=output,in=view,dataType=[]*Report" view:"report,type=Report,table=reports,limit=500,selectorLimit=true,selectorOffset=true" sql:"uri=studio_reports_reader_report:sql/report.sql"`
	PageLimit  int       `parameter:"PageLimit,kind=output,in=body,dataType=int" json:"limit"`
	PageOffset int       `parameter:"PageOffset,kind=output,in=body,dataType=int" json:"offset"`
}
