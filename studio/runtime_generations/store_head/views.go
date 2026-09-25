package store_head

// GenerationHead is generated canonical view metadata for head.
type GenerationHead struct {
	MaxGeneration int64 `sqlx:"max_generation"`
}
