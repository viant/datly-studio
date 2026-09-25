package store_delete

// StoredPublication is generated canonical view metadata for publication.
type StoredPublication struct {
	ReportId          string                `sqlx:"report_id,primaryKey"`
	DesiredGeneration *int64                `writer:"concurrency" sqlx:"desired_generation"`
	PublicationStatus string                `sqlx:"publication_status"`
	ShouldDelete      bool                  `sqlx:"-" writer:"delete"`
	Has               *StoredPublicationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredPublicationHas"`
}

type StoredPublicationHas struct {
	ReportId          bool
	DesiredGeneration bool
	PublicationStatus bool
	ShouldDelete      bool
}

// CurrentPublicationView is generated canonical view metadata for publication.
type CurrentPublicationView struct {
	ReportId          string `sqlx:"report_id,primaryKey"`
	DesiredGeneration *int64 `sqlx:"desired_generation"`
	PublicationStatus string `sqlx:"publication_status"`
}

type PublicationKeysRow struct {
	ReportId string `sqlx:"report_id"`
}
