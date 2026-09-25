package store_one

// Output is the generated output scaffold for acl.
type Output struct {
	Access []*StoredACL `parameter:"Access,kind=output,in=view,dataType=[]*StoredACL" view:"acl,type=StoredACL,table=report_acl,limit=1" sql:"uri=studio_report_acl_store_one_acl:sql/read.sql"`
}
