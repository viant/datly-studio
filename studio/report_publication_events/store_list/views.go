package store_list

import (
	time "time"
)

// StoredEvent is generated canonical view metadata for event.
type StoredEvent struct {
	EventId        string    `sqlx:"event_id"`
	ReportId       string    `sqlx:"report_id"`
	OwnerId        string    `sqlx:"owner_id"`
	Operation      string    `sqlx:"operation"`
	VersionNo      *int      `sqlx:"version_no"`
	GenerationNo   *int64    `sqlx:"generation_no"`
	Status         string    `sqlx:"status"`
	RequestedBy    string    `sqlx:"requested_by"`
	Reason         *string   `sqlx:"reason"`
	FailureCode    *string   `sqlx:"failure_code"`
	FailureMessage *string   `sqlx:"failure_message"`
	OccurredAt     time.Time `sqlx:"occurred_at"`
}
