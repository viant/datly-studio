package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
)

// WarmupRunInput is the generated input scaffold for warmup_run.
type WarmupRunInput struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.WarmupRead"`
	ReportID string `parameter:"ReportID,kind=query,in=reportId,dataType=string,required=false" predicate:"equal,group=1,w,report_id"`
	VersionNo int `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=false" predicate:"equal,group=1,w,version_no"`
	RunID string `parameter:"RunID,kind=query,in=runId,dataType=string,required=false" predicate:"equal,group=1,w,run_id"`
	Status string `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,w,status"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=warmupRun"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=warmupRun"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=warmupRun"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=warmupRun"`
	Has *WarmupRunInputHas `setMarker:"true" typeName:"WarmupRunInputHas" json:"-" sqlx:"-"`
}

type WarmupRunInputHas struct {
	Jwt bool
	Auth bool
	ReportID bool
	VersionNo bool
	RunID bool
	Status bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
