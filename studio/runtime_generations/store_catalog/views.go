package store_catalog

import (
	time "time"
)

// CatalogGeneration is generated canonical view metadata for generation.
type CatalogGeneration struct {
	GenerationNo   int64     `sqlx:"generation_no"`
	Status         string    `sqlx:"status"`
	SourceRevision string    `sqlx:"source_revision"`
	RequestedAt    time.Time `sqlx:"requested_at"`
}
