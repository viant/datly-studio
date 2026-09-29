package store_active_others

// Input is the generated input scaffold for generation.
type Input struct {
	NamespaceId       string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=false"`
	ExcludeGeneration int64     `parameter:"ExcludeGeneration,kind=query,in=excludeGeneration,dataType=int64,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/runtime_generations/otheractivepredicate.OtherActive"`
	Has               *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NamespaceId       bool
	ExcludeGeneration bool
}
