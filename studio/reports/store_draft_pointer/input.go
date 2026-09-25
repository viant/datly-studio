package store_draft_pointer

// Input is the generated input scaffold for report.
type Input struct {
	Reports                   []*DraftPointer           `parameter:"Reports,kind=body,in=data,dataType=[]*DraftPointer" view:"report,type=DraftPointer,entityHooks=DraftPointerRules,table=reports,selectorNoLimit=true" sql:"uri=studio_reports_store_draft_pointer_report:sql/read.sql"`
	ReportKeys                []ReportKeysRow           `parameter:"ReportKeys,kind=param,in=Reports,cardinality=Many" codec:"structql,'uri=studio_reports_store_draft_pointer_report:sql/report_keys.sql'"`
	CurrentReport             []*CurrentReportView      `parameter:"CurrentReport,kind=view,in=CurrentReport,cardinality=Many" view:"CurrentReport,table=reports,selectorNoLimit=true" sql:"uri=studio_reports_store_draft_pointer_report:sql/current_report.sql"`
	_reportHandlerReadIndexes *ReportHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                       *InputHas                 `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Reports       bool
	ReportKeys    bool
	CurrentReport bool
}
