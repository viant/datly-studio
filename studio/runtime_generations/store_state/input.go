package store_state

// Input is the generated input scaffold for generation.
type Input struct {
	Operation                     string                        `parameter:"Operation,kind=query,in=operation,dataType=string,required=true"`
	TargetGeneration              int64                         `parameter:"TargetGeneration,kind=query,in=targetGeneration,dataType=int64,required=true"`
	Generations                   []*StoredGeneration           `parameter:"Generations,kind=body,in=data,dataType=[]*StoredGeneration" view:"generation,type=StoredGeneration,entityHooks=GenerationStateRules,table=runtime_generations" sql:"uri=studio_runtime_generations_store_state_generation:sql/read.sql"`
	GenerationKeys                []GenerationKeysRow           `parameter:"GenerationKeys,kind=param,in=Generations,cardinality=Many" codec:"structql,'uri=studio_runtime_generations_store_state_generation:sql/generation_keys.sql'"`
	CurrentGeneration             []*CurrentGenerationView      `parameter:"CurrentGeneration,kind=view,in=CurrentGeneration,cardinality=Many" view:"CurrentGeneration,table=runtime_generations" sql:"uri=studio_runtime_generations_store_state_generation:sql/current_generation.sql"`
	_generationHandlerReadIndexes *GenerationHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                           *InputHas                     `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Operation         bool
	TargetGeneration  bool
	Generations       bool
	GenerationKeys    bool
	CurrentGeneration bool
}
