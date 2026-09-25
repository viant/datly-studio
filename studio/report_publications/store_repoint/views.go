package store_repoint

// StoredPublication is generated canonical view metadata for publication.
type StoredPublication struct {
	ReportId          string                `sqlx:"report_id,primaryKey"`
	DesiredGeneration *int64                `sqlx:"desired_generation"`
	ActiveGeneration  *int64                `writer:"concurrency" sqlx:"active_generation"`
	PublicationStatus string                `sqlx:"publication_status"`
	Has               *StoredPublicationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredPublicationHas"`
}

type StoredPublicationHas struct {
	ReportId          bool
	DesiredGeneration bool
	ActiveGeneration  bool
	PublicationStatus bool
}

// CurrentPublicationView is generated canonical view metadata for publication.
type CurrentPublicationView struct {
	ReportId          string `sqlx:"report_id,primaryKey"`
	DesiredGeneration *int64 `sqlx:"desired_generation"`
	ActiveGeneration  *int64 `sqlx:"active_generation"`
	PublicationStatus string `sqlx:"publication_status"`
}

type PublicationKeysRow struct {
	ReportId string `sqlx:"report_id"`
}
