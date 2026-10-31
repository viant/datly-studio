package value

import (
	contract "github.com/viant/forge/backend/artifactrelease"
	"time"
)

// HistoryEntry records an attempted revision; only applied entries are active.
type HistoryEntry struct {
	OperationID string
	Next        contract.Channel
	Status      string
	CreatedAt   time.Time
	CreatedBy   string
	CompletedAt *time.Time
	FailureCode string
}
