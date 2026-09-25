package store_run_access

// Output is the generated output scaffold for access.
type Output struct {
	Allowed []*RunAccess `parameter:"Allowed,kind=output,in=view,dataType=[]*RunAccess" view:"access,type=RunAccess,table=reports,limit=1" sql:"uri=studio_reports_store_run_access_access:sql/read.sql"`
}
