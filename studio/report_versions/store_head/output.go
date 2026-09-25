package store_head

// Output is the generated output scaffold for head.
type Output struct {
	Heads []*VersionHead `parameter:"Heads,kind=output,in=view,dataType=[]*VersionHead" view:"head,type=VersionHead,table=report_versions,limit=1" sql:"uri=studio_report_versions_store_head_head:sql/read.sql"`
}
