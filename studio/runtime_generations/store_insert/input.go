package store_insert

// Input is the generated input scaffold for generation.
type Input struct {
	Generations []*StoredGeneration `parameter:"Generations,kind=body,in=data,dataType=[]*StoredGeneration" view:"generation,type=StoredGeneration,entityHooks=GenerationInsertRules,table=runtime_generations" sql:"uri=studio_runtime_generations_store_insert_generation:sql/read.sql"`
	Has         *InputHas           `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Generations bool
}
