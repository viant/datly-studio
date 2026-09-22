package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for exposure.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Exposures []*ReportMCPExposure `parameter:"Exposures,kind=body,in=data,dataType=[]*ReportMCPExposure" view:"exposure,type=ReportMCPExposure,entityHooks=ExposureRules,table=report_mcp_exposures" sql:"uri=studio_report_mcp_exposures_writer_exposure:sql/exposure.sql"`
	ExposureKeys []ExposureKeysRow `parameter:"ExposureKeys,kind=param,in=Exposures,cardinality=Many" codec:"structql,'uri=studio_report_mcp_exposures_writer_exposure:sql/exposure_keys.sql'"`
	CurrentExposure []*CurrentExposureView `parameter:"CurrentExposure,kind=view,in=CurrentExposure,cardinality=Many" view:"CurrentExposure,table=report_mcp_exposures" sql:"uri=studio_report_mcp_exposures_writer_exposure:sql/current_exposure.sql"`
	_exposureHandlerReadIndexes *ExposureHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Exposures bool
	ExposureKeys bool
	CurrentExposure bool
}
