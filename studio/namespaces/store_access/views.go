package store_access

// StoredNamespace is generated canonical view metadata for namespace.
type StoredNamespace struct {
	OwnerId string `sqlx:"owner_id"`
	Name    string `sqlx:"name"`
}
