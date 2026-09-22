package reader

import (
	time "time"
)

// SessionMetadata is generated canonical view metadata for session.
type SessionMetadata struct {
	SessionIdHash any        `sqlx:"session_id_hash"`
	SubjectId     any        `sqlx:"subject_id"`
	ExpiresAtUnix *int       `sqlx:"expires_at_unix"`
	CreatedAt     *time.Time `sqlx:"created_at"`
}
