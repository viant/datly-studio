package store_capabilities

// Output is the generated output scaffold for capability.
type Output struct {
	Capabilities []*ReportCapability `parameter:"Capabilities,kind=output,in=view,dataType=[]*ReportCapability" view:"capability,type=ReportCapability,table=components,limit=1" sql:"uri=studio_reports_store_capabilities_capability:sql/read.sql"`
}
