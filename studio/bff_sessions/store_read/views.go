package store_read

import (
	time "time"
)

// StoredSession is generated canonical view metadata for session.
type StoredSession struct {
	SessionIdHash     string     `sqlx:"session_id_hash"`
	SubjectId         string     `sqlx:"subject_id"`
	PayloadCiphertext []byte     `sqlx:"payload_ciphertext"`
	ExpiresAtUnix     int64      `sqlx:"expires_at_unix"`
	CreatedAt         *time.Time `sqlx:"created_at"`
}
