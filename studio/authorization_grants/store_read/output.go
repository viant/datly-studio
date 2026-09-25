package store_read

// Output is the generated output scaffold for grant.
type Output struct {
	Grants []*Grant `parameter:"Grants,kind=output,in=view,dataType=[]*Grant" view:"grant,type=Grant,table=reports,limit=2" sql:"uri=studio_authorization_grants_store_read_grant:sql/read.sql"`
}
