package store_config

// Input is the generated input scaffold for report.
type Input struct {
	Reports                   []*StoredReport           `parameter:"Reports,kind=body,in=data,dataType=[]*StoredReport" view:"report,type=StoredReport,entityHooks=ReportConfigRules,table=reports" sql:"uri=studio_reports_store_config_report:sql/read.sql"`
	ReportKeys                []ReportKeysRow           `parameter:"ReportKeys,kind=param,in=Reports,cardinality=Many" codec:"structql,'uri=studio_reports_store_config_report:sql/report_keys.sql'"`
	CurrentReport             []*CurrentReportView      `parameter:"CurrentReport,kind=view,in=CurrentReport,cardinality=Many" view:"CurrentReport,table=reports" sql:"uri=studio_reports_store_config_report:sql/current_report.sql"`
	_reportHandlerReadIndexes *ReportHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                       *InputHas                 `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Reports       bool
	ReportKeys    bool
	CurrentReport bool
}
