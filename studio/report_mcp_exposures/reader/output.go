package reader

// Output is the generated output scaffold for exposure.
type Output struct {
	Exposures []*ReportMCPExposure `parameter:"Exposures,kind=output,in=view,dataType=[]*ReportMCPExposure" view:"exposure,type=ReportMCPExposure,table=report_mcp_exposures,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={route_id,route_method,route_path,kind,name,enabled,ordinal}" sql:"uri=studio_report_mcp_exposures_reader_exposure:sql/exposure.sql"`
}
