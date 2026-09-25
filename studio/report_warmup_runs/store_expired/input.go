package store_expired

import (
	time "time"
)

// Input is the generated input scaffold for warmup_run.
type Input struct {
	Before    time.Time `parameter:"Before,kind=query,in=before,dataType=time.Time,required=true"`
	PageLimit int       `parameter:"PageLimit,kind=query,in=pageLimit,dataType=int,required=true"`
	Has       *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Before    bool
	PageLimit bool
}
