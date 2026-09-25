package store_active_others

// Input is the generated input scaffold for generation.
type Input struct {
	ExcludeGeneration int64     `parameter:"ExcludeGeneration,kind=query,in=excludeGeneration,dataType=int64,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/runtime_generations/otheractivepredicate.OtherActive"`
	Has               *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ExcludeGeneration bool
}
