package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// WarmupGetInput is the generated input scaffold for warmup_run.
type WarmupGetInput struct {
	Jwt      *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.WarmupRead"`
	ReportId string             `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId" predicate:"equal,group=2,w,report_id"`
	RunId    string             `parameter:"RunId,kind=body,in=runId,dataType=string,required=true" json:"runId" predicate:"equal,group=2,w,run_id"`
	Has      *WarmupGetInputHas `setMarker:"true" typeName:"WarmupGetInputHas" json:"-" sqlx:"-"`
}

type WarmupGetInputHas struct {
	Jwt      bool
	Auth     bool
	ReportId bool
	RunId    bool
}
