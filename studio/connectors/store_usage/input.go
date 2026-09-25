package store_usage

// Input is the generated input scaffold for usage.
type Input struct {
	Name string    `parameter:"Name,kind=query,in=name,dataType=string,required=true"`
	Has  *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Name bool
}
