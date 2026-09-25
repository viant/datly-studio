package store_staged

// StagedPublication is generated canonical view metadata for publication.
type StagedPublication struct {
	ReportId          string `sqlx:"report_id"`
	DesiredGeneration int64  `sqlx:"desired_generation"`
	PublicationStatus string `sqlx:"publication_status"`
}
