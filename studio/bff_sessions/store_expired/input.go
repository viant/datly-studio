package store_expired

// Input is the generated input scaffold for session.
type Input struct {
	ExpiresBefore int64     `parameter:"ExpiresBefore,kind=query,in=expiresBefore,dataType=int64,required=true"`
	Has           *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ExpiresBefore bool
}
