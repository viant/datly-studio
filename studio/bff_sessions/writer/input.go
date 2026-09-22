package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for session.
type Input struct {
	Jwt                        *jwt.Claims                `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Sessions                   []*SessionRevocation       `parameter:"Sessions,kind=body,in=data,dataType=[]*SessionRevocation" view:"session,type=SessionRevocation,entityHooks=SessionRevocationRules,table=bff_sessions" sql:"uri=studio_bff_sessions_writer_session:sql/patch.sql"`
	SessionKeys                []SessionKeysRow           `parameter:"SessionKeys,kind=param,in=Sessions,cardinality=Many" codec:"structql,'uri=studio_bff_sessions_writer_session:sql/session_keys.sql'"`
	CurrentSession             []*CurrentSessionView      `parameter:"CurrentSession,kind=view,in=CurrentSession,cardinality=Many" view:"CurrentSession,table=bff_sessions" sql:"uri=studio_bff_sessions_writer_session:sql/current_session.sql"`
	_sessionHandlerReadIndexes *SessionHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                        *InputHas                  `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt            bool
	Sessions       bool
	SessionKeys    bool
	CurrentSession bool
}
