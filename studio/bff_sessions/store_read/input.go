package store_read

// Input is the generated input scaffold for session.
type Input struct {
	SessionIdHash string    `parameter:"SessionIdHash,kind=query,in=sessionIdHash,dataType=string,required=true" predicate:"equal,s,session_id_hash"`
	Has           *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	SessionIdHash bool
}
