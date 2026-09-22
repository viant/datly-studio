package reader

// Output is the generated output scaffold for config.
type Output struct {
	Configs []*ReportCubeConfig `parameter:"Configs,kind=output,in=view,dataType=[]*ReportCubeConfig" view:"config,type=ReportCubeConfig,table=report_cube_configs" sql:"uri=studio_report_cube_configs_reader_config:sql/config.sql"`
}
