package store_expired

// Output is the generated output scaffold for session.
type Output struct {
	Sessions []*ExpiredSession `parameter:"Sessions,kind=output,in=view,dataType=[]*ExpiredSession" view:"session,type=ExpiredSession,table=bff_sessions,limit=256" sql:"uri=studio_bff_sessions_store_expired_session:sql/read.sql"`
}
