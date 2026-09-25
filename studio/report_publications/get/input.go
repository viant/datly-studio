package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// PublicationGetInput is the generated input scaffold for publication.
type PublicationGetInput struct {
	Jwt      *jwt.Claims             `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output      `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.PublicationRead"`
	ReportId string                  `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId" predicate:"equal,group=2,p,report_id"`
	Has      *PublicationGetInputHas `setMarker:"true" typeName:"PublicationGetInputHas" json:"-" sqlx:"-"`
}

type PublicationGetInputHas struct {
	Jwt      bool
	Auth     bool
	ReportId bool
}
