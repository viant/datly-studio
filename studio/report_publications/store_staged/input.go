package store_staged

// Input is the generated input scaffold for publication.
type Input struct {
	DesiredGeneration int64     `parameter:"DesiredGeneration,kind=query,in=desiredGeneration,dataType=int64,required=true" predicate:"equal,group=1,p,desired_generation"`
	Has               *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	DesiredGeneration bool
}
