package list

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	listoptions "github.com/viant/datly-studio/studio/report_versions/listoptions"
	jwt "github.com/viant/scy/auth/jwt"
)

// VersionListInput is the generated input scaffold for version.
type VersionListInput struct {
	Jwt      *jwt.Claims          `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output   `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportVersionMetadataRead"`
	ReportId string               `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	Input    listoptions.Options  `parameter:"Input,kind=body,in=input,dataType=listoptions.Options,required=false" json:"input"`
	Limit    int                  `parameter:"Limit,kind=body,in=limit,dataType=int,required=false,cacheable=false" json:"-" querySelector:"view=version"`
	Offset   int                  `parameter:"Offset,kind=body,in=offset,dataType=int,required=false,cacheable=false" json:"-" querySelector:"view=version"`
	Has      *VersionListInputHas `setMarker:"true" typeName:"VersionListInputHas" json:"-" sqlx:"-"`
}

type VersionListInputHas struct {
	Jwt      bool
	Auth     bool
	ReportId bool
	Input    bool
	Limit    bool
	Offset   bool
}
