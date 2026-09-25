package store_expired

// ExpiredSession is generated canonical view metadata for session.
type ExpiredSession struct {
	SessionIdHash string `sqlx:"session_id_hash"`
}
