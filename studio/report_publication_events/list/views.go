package list

import (
	time "time"
)

// PublicationEvent is generated canonical view metadata for event.
type PublicationEvent struct {
	EventId        string    `sqlx:"event_id"`
	ReportId       string    `sqlx:"report_id"`
	OwnerId        string    `sqlx:"owner_id"`
	Operation      string    `sqlx:"operation"`
	VersionNo      *int      `json:"versionNo,omitempty" sqlx:"version_no"`
	GenerationNo   *int64    `json:"generationNo,omitempty" sqlx:"generation_no"`
	Status         string    `sqlx:"status"`
	RequestedBy    string    `sqlx:"requested_by"`
	Reason         *string   `json:"reason,omitempty" sqlx:"reason"`
	FailureCode    *string   `json:"failureCode,omitempty" sqlx:"failure_code"`
	FailureMessage *string   `json:"failureMessage,omitempty" sqlx:"failure_message"`
	OccurredAt     time.Time `sqlx:"occurred_at"`
}
