package store_presence

// Output is the generated output scaffold for usage.
type Output struct {
	Usages []*NamespaceUsage `parameter:"Usages,kind=output,in=view,dataType=[]*NamespaceUsage" view:"usage,type=NamespaceUsage,table=report_resource_files,limit=1" sql:"uri=studio_resource_namespaces_store_presence_usage:sql/read.sql"`
}
