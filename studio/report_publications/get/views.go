package get

import (
	time "time"
)

// Publication is generated canonical view metadata for publication.
type Publication struct {
	ReportId          string     `sqlx:"report_id,refTable=report_versions,refColumn=report_id,required=true,primaryKey=true"`
	ActiveVersionNo   int        `sqlx:"active_version_no,refTable=report_versions,refColumn=version_no,required=true"`
	DesiredVersionNo  *int       `sqlx:"desired_version_no,refTable=report_versions,refColumn=version_no"`
	DesiredGeneration int64      `sqlx:"desired_generation,required=true"`
	ActiveGeneration  *int64     `sqlx:"active_generation,refTable=runtime_generations,refColumn=generation_no"`
	PublicationStatus string     `sqlx:"publication_status,required=true"`
	RuntimeRevision   *string    `sqlx:"runtime_revision"`
	SpecHash          *string    `sqlx:"spec_hash,required=true"`
	PublishedAt       *time.Time `sqlx:"published_at,required=true"`
}
