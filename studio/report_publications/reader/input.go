package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for publication.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,p,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.PublicationRead"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
}
