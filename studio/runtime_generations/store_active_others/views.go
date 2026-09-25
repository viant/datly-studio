package store_active_others

// ActiveGeneration is generated canonical view metadata for generation.
type ActiveGeneration struct {
	GenerationNo int64  `sqlx:"generation_no"`
	Status       string `sqlx:"status"`
}
