package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for exposure.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId string `parameter:"ReportId,kind=path,in=reportId,dataType=string,required=true" predicate:"equal,group=2,e,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportMCPRead"`
	VersionNo int `parameter:"VersionNo,kind=path,in=versionNo,dataType=int,required=true" predicate:"equal,group=2,e,version_no"`
	Kind string `parameter:"Kind,kind=query,in=kind,dataType=string,required=false" predicate:"equal,group=1,e,kind"`
	RouteId string `parameter:"RouteId,kind=query,in=routeId,dataType=string,required=false" predicate:"equal,group=1,e,route_id"`
	Enabled bool `parameter:"Enabled,kind=query,in=enabled,dataType=bool,required=false" predicate:"equal,group=1,e,enabled"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=exposure"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=exposure"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=exposure"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=exposure"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	ReportId bool
	VersionNo bool
	Kind bool
	RouteId bool
	Enabled bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
