package store_usage

// Input is the generated input scaffold for usage.
type Input struct {
	OwnerId string    `parameter:"OwnerId,kind=query,in=ownerId,dataType=string,required=true"`
	Name    string    `parameter:"Name,kind=query,in=name,dataType=string,required=true"`
	Has     *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	OwnerId bool
	Name    bool
}
