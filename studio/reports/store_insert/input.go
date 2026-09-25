package store_insert

// Input is the generated input scaffold for report.
type Input struct {
	Reports []*StoredReport `parameter:"Reports,kind=body,in=data,dataType=[]*StoredReport" view:"report,type=StoredReport,entityHooks=ReportInsertRules,table=reports" sql:"uri=studio_reports_store_insert_report:sql/read.sql"`
	Has     *InputHas       `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Reports bool
}
