package writer

// ReportMCPExposure is generated canonical view metadata for exposure.
type ReportMCPExposure struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ExposureId *string `sqlx:"exposure_id,primaryKey,required=true" validate:"required"`
	RouteId *string `validate:"required" sqlx:"route_id,required=true"`
	RouteMethod *string `validate:"required" sqlx:"route_method,required=true"`
	RoutePath *string `validate:"required" sqlx:"route_path,required=true"`
	Kind *string `validate:"required,choice(tool,resource,resourceTemplate)" sqlx:"kind,required=true"`
	Name *string `validate:"required" sqlx:"name,required=true"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	Description *string `sqlx:"description"`
	DescriptionPath *string `sqlx:"description_path"`
	MimeType *string `sqlx:"mime_type"`
	Enabled *int `sqlx:"enabled,required=true"`
	Ordinal *int `sqlx:"ordinal,required=true"`
	Has *ReportMCPExposureHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportMCPExposureHas"`
}

type ReportMCPExposureHas struct {
	ReportId bool
	VersionNo bool
	ExposureId bool
	RouteId bool
	RouteMethod bool
	RoutePath bool
	Kind bool
	Name bool
	ShouldDelete bool
	Description bool
	DescriptionPath bool
	MimeType bool
	Enabled bool
	Ordinal bool
}

// CurrentExposureView is generated canonical view metadata for exposure.
type CurrentExposureView struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ExposureId *string `sqlx:"exposure_id,primaryKey,required=true" validate:"required"`
	RouteId *string `validate:"required" sqlx:"route_id,required=true"`
	RouteMethod *string `validate:"required" sqlx:"route_method,required=true"`
	RoutePath *string `validate:"required" sqlx:"route_path,required=true"`
	Kind *string `validate:"required,choice(tool,resource,resourceTemplate)" sqlx:"kind,required=true"`
	Name *string `validate:"required" sqlx:"name,required=true"`
	Description *string `sqlx:"description"`
	DescriptionPath *string `sqlx:"description_path"`
	MimeType *string `sqlx:"mime_type"`
	Enabled *int `sqlx:"enabled,required=true"`
	Ordinal *int `sqlx:"ordinal,required=true"`
}

type ExposureKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ExposureId *string `sqlx:"exposure_id"`
}
