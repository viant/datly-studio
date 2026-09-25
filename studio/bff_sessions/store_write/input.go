package store_write

// Input is the generated input scaffold for session.
type Input struct {
	Sessions                   []*StoredSession           `parameter:"Sessions,kind=body,in=data,dataType=[]*StoredSession" view:"session,type=StoredSession,table=bff_sessions" sql:"uri=studio_bff_sessions_store_write_session:sql/patch.sql"`
	SessionKeys                []SessionKeysRow           `parameter:"SessionKeys,kind=param,in=Sessions,cardinality=Many" codec:"structql,'uri=studio_bff_sessions_store_write_session:sql/session_keys.sql'"`
	CurrentSession             []*CurrentSessionView      `parameter:"CurrentSession,kind=view,in=CurrentSession,cardinality=Many" view:"CurrentSession,table=bff_sessions" sql:"uri=studio_bff_sessions_store_write_session:sql/current_session.sql"`
	_sessionHandlerReadIndexes *SessionHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                        *InputHas                  `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Sessions       bool
	SessionKeys    bool
	CurrentSession bool
}
