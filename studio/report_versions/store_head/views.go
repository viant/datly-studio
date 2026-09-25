package store_head

// VersionHead is generated canonical view metadata for head.
type VersionHead struct {
	MaxVersionNo int `sqlx:"max_version_no"`
}
