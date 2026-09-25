package store_state

import (
	time "time"
)

// StoredGeneration is generated canonical view metadata for generation.
type StoredGeneration struct {
	GenerationNo int64                `sqlx:"generation_no,primaryKey"`
	Status       string               `writer:"concurrency" sqlx:"status"`
	ReportCount  *int                 `sqlx:"report_count"`
	ActivatedAt  *time.Time           `sqlx:"activated_at"`
	RetiredAt    *time.Time           `sqlx:"retired_at"`
	Has          *StoredGenerationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredGenerationHas"`
}

type StoredGenerationHas struct {
	GenerationNo bool
	Status       bool
	ReportCount  bool
	ActivatedAt  bool
	RetiredAt    bool
}

// CurrentGenerationView is generated canonical view metadata for generation.
type CurrentGenerationView struct {
	GenerationNo int64      `sqlx:"generation_no,primaryKey"`
	Status       string     `sqlx:"status"`
	ReportCount  *int       `sqlx:"report_count"`
	ActivatedAt  *time.Time `sqlx:"activated_at"`
	RetiredAt    *time.Time `sqlx:"retired_at"`
}

type GenerationKeysRow struct {
	GenerationNo int64 `sqlx:"generation_no"`
}
