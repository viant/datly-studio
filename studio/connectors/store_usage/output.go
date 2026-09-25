package store_usage

// Output is the generated output scaffold for usage.
type Output struct {
	Usages []*ConnectorUsage `parameter:"Usages,kind=output,in=view,dataType=[]*ConnectorUsage" view:"usage,type=ConnectorUsage,table=reports,limit=1" sql:"uri=studio_connectors_store_usage_usage:sql/read.sql"`
}
