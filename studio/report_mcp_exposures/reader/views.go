package reader

// ReportMCPExposure is generated canonical view metadata for exposure.
type ReportMCPExposure struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ExposureId *string `sqlx:"exposure_id"`
	RouteId *string `sqlx:"route_id"`
	RouteMethod *string `sqlx:"route_method"`
	RoutePath *string `sqlx:"route_path"`
	Kind *string `sqlx:"kind"`
	Name *string `sqlx:"name"`
	Description *string `sqlx:"description"`
	DescriptionPath *string `sqlx:"description_path"`
	MimeType *string `sqlx:"mime_type"`
	Enabled *int `sqlx:"enabled"`
	Ordinal *int `sqlx:"ordinal"`
}
