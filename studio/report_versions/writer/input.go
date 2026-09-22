package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for version.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Versions []*ReportVersion `parameter:"Versions,kind=body,in=data,dataType=[]*ReportVersion" view:"version,type=ReportVersion,entityHooks=VersionRules,table=report_versions" sql:"uri=studio_report_versions_writer_version:sql/version.sql"`
	VersionKeys []VersionKeysRow `parameter:"VersionKeys,kind=param,in=Versions,cardinality=Many" codec:"structql,'uri=studio_report_versions_writer_version:sql/version_keys.sql'"`
	CurrentVersion []*CurrentVersionView `parameter:"CurrentVersion,kind=view,in=CurrentVersion,cardinality=Many" view:"CurrentVersion,table=report_versions" sql:"uri=studio_report_versions_writer_version:sql/current_version.sql"`
	_versionHandlerReadIndexes *VersionHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Versions bool
	VersionKeys bool
	CurrentVersion bool
}
