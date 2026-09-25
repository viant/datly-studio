package store_read

// Output is the generated output scaffold for session.
type Output struct {
	Sessions []*StoredSession `parameter:"Sessions,kind=output,in=view,dataType=[]*StoredSession" view:"session,type=StoredSession,table=bff_sessions,limit=1" sql:"uri=studio_bff_sessions_store_read_session:sql/read.sql"`
}
