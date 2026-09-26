package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// VersionGetInput is the generated input scaffold for version.
type VersionGetInput struct {
	Jwt       *jwt.Claims         `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output  `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportVersionMetadataRead"`
	ReportId  string              `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId" predicate:"equal,group=2,v,report_id"`
	VersionNo int                 `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo" predicate:"equal,group=2,v,version_no"`
	Has       *VersionGetInputHas `setMarker:"true" typeName:"VersionGetInputHas" json:"-" sqlx:"-"`
}

type VersionGetInputHas struct {
	Jwt       bool
	Auth      bool
	ReportId  bool
	VersionNo bool
}
