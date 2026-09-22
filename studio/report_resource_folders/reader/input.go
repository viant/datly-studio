package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for folder.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,f,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportResourceFolderRead"`
	VersionNo int `parameter:"VersionNo,kind=path,in=versionNo,dataType=int,required=true" predicate:"equal,group=2,f,version_no"`
	Namespace string `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=false" predicate:"equal,group=1,f,namespace"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
	VersionNo bool
	Namespace bool
}
