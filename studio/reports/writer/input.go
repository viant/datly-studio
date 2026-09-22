package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for report.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Reports []*Report `parameter:"Reports,kind=body,in=data,dataType=[]*Report" view:"report,type=Report,table=reports" sql:"uri=studio_reports_writer_report:sql/report.sql"`
	ReportKeys []ReportKeysRow `parameter:"ReportKeys,kind=param,in=Reports,cardinality=Many" codec:"structql,'uri=studio_reports_writer_report:sql/report_keys.sql'"`
	CurrentReport []*CurrentReportView `parameter:"CurrentReport,kind=view,in=CurrentReport,cardinality=Many" view:"CurrentReport,table=reports" sql:"uri=studio_reports_writer_report:sql/current_report.sql"`
	_reportHandlerReadIndexes *ReportHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Reports bool
	ReportKeys bool
	CurrentReport bool
}
