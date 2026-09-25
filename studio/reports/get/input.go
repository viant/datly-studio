package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// ReportGetInput is the generated input scaffold for report.
type ReportGetInput struct {
	Jwt  *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/reports/catalogpredicate.ReportCatalogRead"`
	Id   string             `parameter:"Id,kind=body,in=id,dataType=string,required=true" json:"id" predicate:"equal,group=2,r,id"`
	Has  *ReportGetInputHas `setMarker:"true" typeName:"ReportGetInputHas" json:"-" sqlx:"-"`
}

type ReportGetInputHas struct {
	Jwt  bool
	Auth bool
	Id   bool
}
