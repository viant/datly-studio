package reader

// Output is the generated output scaffold for acl.
type Output struct {
	Items []*ReportACL `parameter:"Items,kind=output,in=view,dataType=[]*ReportACL" view:"acl,type=ReportACL,table=report_acl" sql:"uri=studio_report_acl_reader_acl:sql/acl.sql"`
}
