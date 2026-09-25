package store_usage

// Output is the generated output scaffold for usage.
type Output struct {
	Usages []*NamespaceUsage `parameter:"Usages,kind=output,in=view,dataType=[]*NamespaceUsage" view:"usage,type=NamespaceUsage,table=reports,limit=1" sql:"uri=studio_namespaces_store_usage_usage:sql/read.sql"`
}
