package reader

// Output is the generated output scaffold for parameter.
type Output struct {
	Parameters []*ReportParameter `parameter:"Parameters,kind=output,in=view,dataType=[]*ReportParameter" view:"parameter,type=ReportParameter,table=report_parameters" sql:"uri=studio_report_parameters_reader_parameter:sql/parameter.sql"`
}
