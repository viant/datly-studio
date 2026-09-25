package reader

import (
	time "time"
)

// SessionMetadata is generated canonical view metadata for session.
type SessionMetadata struct {
	SessionIdHash string     `sqlx:"session_id_hash"`
	SubjectId     string     `sqlx:"subject_id"`
	ExpiresAtUnix *int       `sqlx:"expires_at_unix"`
	CreatedAt     *time.Time `sqlx:"created_at"`
}
