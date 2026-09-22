package writer

// SessionRevocation is generated canonical view metadata for session.
type SessionRevocation struct {
	SessionIdHash string                `sqlx:"session_id_hash,primaryKey" validate:"required"`
	SubjectId     string                `validate:"required" sqlx:"subject_id"`
	ExpiresAtUnix *int                  `validate:"required" sqlx:"expires_at_unix"`
	Has           *SessionRevocationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"SessionRevocationHas"`
}

type SessionRevocationHas struct {
	SessionIdHash bool
	SubjectId     bool
	ExpiresAtUnix bool
}

// CurrentSessionView is generated canonical view metadata for session.
type CurrentSessionView struct {
	SessionIdHash string `sqlx:"session_id_hash,primaryKey" validate:"required"`
	SubjectId     string `validate:"required" sqlx:"subject_id"`
	ExpiresAtUnix *int   `validate:"required" sqlx:"expires_at_unix"`
}

type SessionKeysRow struct {
	SessionIdHash string `sqlx:"session_id_hash"`
}
