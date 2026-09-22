package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for parameter.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,p,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportParameterRead"`
	VersionNo int `parameter:"VersionNo,kind=path,in=versionNo,dataType=int,required=true" predicate:"equal,group=2,p,version_no"`
	SourceKind string `parameter:"SourceKind,kind=query,in=sourceKind,dataType=string,required=false" predicate:"equal,group=1,p,source_kind"`
	Required bool `parameter:"Required,kind=query,in=required,dataType=bool,required=false" predicate:"equal,group=1,p,required"`
	EmitOutput bool `parameter:"EmitOutput,kind=query,in=emitOutput,dataType=bool,required=false" predicate:"equal,group=1,p,emit_output"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=parameter"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=parameter"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=parameter"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=parameter"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
	VersionNo bool
	SourceKind bool
	Required bool
	EmitOutput bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
