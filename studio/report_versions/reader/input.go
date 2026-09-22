package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for version.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,v,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportVersionRead"`
	VersionNo int `parameter:"VersionNo,kind=path,in=versionNo,dataType=int,uri=/{versionNo},required=true" predicate:"equal,group=2,v,version_no"`
	State string `parameter:"State,kind=query,in=state,dataType=string,required=false" predicate:"equal,group=1,v,state"`
	AuthoringMode string `parameter:"AuthoringMode,kind=query,in=authoringMode,dataType=string,required=false" predicate:"equal,group=1,v,authoring_mode"`
	CompileStatus string `parameter:"CompileStatus,kind=query,in=compileStatus,dataType=string,required=false" predicate:"equal,group=1,v,compile_status"`
	CreatedBy string `parameter:"CreatedBy,kind=query,in=createdBy,dataType=string,required=false" predicate:"equal,group=1,v,created_by"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=version"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=version"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=version"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=version"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
	VersionNo bool
	State bool
	AuthoringMode bool
	CompileStatus bool
	CreatedBy bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
