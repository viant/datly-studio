package store_list

// Output is the generated output scaffold for acl.
type Output struct {
	Access []*StoredACL `parameter:"Access,kind=output,in=view,dataType=[]*StoredACL" view:"acl,type=StoredACL,table=report_acl,selectorNoLimit=true" sql:"uri=studio_report_acl_store_list_acl:sql/read.sql"`
}
