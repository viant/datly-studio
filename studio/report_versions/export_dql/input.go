package export_dql

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// VersionExportInput is the generated input scaffold for version.
type VersionExportInput struct {
	NamespaceId *string                `parameter:"NamespaceId,kind=header,in=X-Studio-Namespace,dataType=*string,required=false" json:"namespaceId,omitempty"`
	Jwt         *jwt.Claims            `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth        *studioauth.Output     `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ReportVersionRead"`
	ReportId    string                 `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId" predicate:"equal,group=2,v,report_id"`
	VersionNo   int                    `parameter:"VersionNo,kind=body,in=versionNo,dataType=int,required=true" json:"versionNo" predicate:"equal,group=2,v,version_no"`
	Has         *VersionExportInputHas `setMarker:"true" typeName:"VersionExportInputHas" json:"-" sqlx:"-"`
}

type VersionExportInputHas struct {
	NamespaceId bool
	Jwt         bool
	Auth        bool
	ReportId    bool
	VersionNo   bool
}
