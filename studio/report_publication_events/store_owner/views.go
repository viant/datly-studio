package store_owner

// StoredReport is generated canonical view metadata for report.
type StoredReport struct {
	Id      string `sqlx:"id"`
	OwnerId string `sqlx:"owner_id"`
}
