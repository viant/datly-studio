package reader

// Output is the generated output scaffold for acl.
type Output struct {
	Access []*ReportACL `parameter:"Access,kind=output,in=view,dataType=[]*ReportACL" view:"acl,type=ReportACL,table=report_acl" sql:"uri=studio_report_acl_reader_acl:sql/acl.sql"`
}
