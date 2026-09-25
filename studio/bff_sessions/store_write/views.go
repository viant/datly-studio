package store_write

import (
	time "time"
)

// StoredSession is generated canonical view metadata for session.
type StoredSession struct {
	SessionIdHash     string            `sqlx:"session_id_hash,primaryKey" validate:"required"`
	SubjectId         string            `sqlx:"subject_id"`
	PayloadCiphertext []byte            `sqlx:"payload_ciphertext"`
	ExpiresAtUnix     int64             `sqlx:"expires_at_unix"`
	ShouldDelete      bool              `sqlx:"-" writer:"delete"`
	CreatedAt         *time.Time        `sqlx:"created_at"`
	Has               *StoredSessionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredSessionHas"`
}

type StoredSessionHas struct {
	SessionIdHash     bool
	SubjectId         bool
	PayloadCiphertext bool
	ExpiresAtUnix     bool
	ShouldDelete      bool
	CreatedAt         bool
}

// CurrentSessionView is generated canonical view metadata for session.
type CurrentSessionView struct {
	SessionIdHash     string     `sqlx:"session_id_hash,primaryKey" validate:"required"`
	SubjectId         string     `sqlx:"subject_id"`
	PayloadCiphertext []byte     `sqlx:"payload_ciphertext"`
	ExpiresAtUnix     int64      `sqlx:"expires_at_unix"`
	CreatedAt         *time.Time `sqlx:"created_at"`
}

type SessionKeysRow struct {
	SessionIdHash string `sqlx:"session_id_hash"`
}
