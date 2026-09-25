package store_active_others

// ActivePublication is generated canonical view metadata for publication.
type ActivePublication struct {
	ReportId          string `sqlx:"report_id"`
	DesiredGeneration int64  `sqlx:"desired_generation"`
	ActiveGeneration  *int64 `sqlx:"active_generation"`
	PublicationStatus string `sqlx:"publication_status"`
}
