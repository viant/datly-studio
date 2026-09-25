package store_status

// Input is the generated input scaffold for generation.
type Input struct {
	Status string    `parameter:"Status,kind=query,in=status,dataType=string,required=true" predicate:"equal,group=1,g,status"`
	Has    *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Status bool
}
