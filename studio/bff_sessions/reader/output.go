package reader

// Output is the generated output scaffold for session.
type Output struct {
	Sessions []*SessionMetadata `parameter:"Sessions,kind=output,in=view,dataType=[]*SessionMetadata" view:"session,type=SessionMetadata,table=bff_sessions,limit=100" sql:"uri=studio_bff_sessions_reader_session:sql/read.sql"`
}
