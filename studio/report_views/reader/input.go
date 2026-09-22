package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for view.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,v,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportViewRead"`
	VersionNo int `parameter:"VersionNo,kind=path,in=versionNo,dataType=int,required=true" predicate:"equal,group=2,v,version_no"`
	Role string `parameter:"Role,kind=query,in=role,dataType=string,required=false" predicate:"equal,group=1,v,role"`
	SourceKind string `parameter:"SourceKind,kind=query,in=sourceKind,dataType=string,required=false" predicate:"equal,group=1,v,source_kind"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=view"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=view"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=view"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=view"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
	VersionNo bool
	Role bool
	SourceKind bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
