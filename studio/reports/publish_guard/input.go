package publish_guard

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for report.
type Input struct {
	Jwt      *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportPublish"`
	ReportId string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId" predicate:"equal,group=2,report,id"`
	Has      *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt      bool
	Auth     bool
	ReportId bool
}
