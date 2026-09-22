package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for skill.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,s,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportSkillRead"`
	VersionNo int `parameter:"VersionNo,kind=path,in=versionNo,dataType=int,required=true" predicate:"equal,group=2,s,version_no"`
	FolderId string `parameter:"FolderId,kind=query,in=folderId,dataType=string,required=false" predicate:"equal,group=1,s,folder_id"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
	VersionNo bool
	FolderId bool
}
