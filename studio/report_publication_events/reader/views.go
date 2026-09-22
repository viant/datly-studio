package reader

import "time"

// PublicationEvent is the non-secret, immutable lifecycle evidence exposed to
// an owner through the Datly control-plane reader.
type PublicationEvent struct {
	EventID        *string    `sqlx:"event_id"`
	ReportID       *string    `sqlx:"report_id"`
	OwnerID        *string    `sqlx:"owner_id"`
	Operation      *string    `sqlx:"operation"`
	VersionNo      *int       `sqlx:"version_no"`
	GenerationNo   *int       `sqlx:"generation_no"`
	Status         *string    `sqlx:"status"`
	RequestedBy    *string    `sqlx:"requested_by"`
	Reason         *string    `sqlx:"reason"`
	FailureCode    *string    `sqlx:"failure_code"`
	FailureMessage *string    `sqlx:"failure_message"`
	OccurredAt     *time.Time `sqlx:"occurred_at"`
}
