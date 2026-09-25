package store_active_others

// Output is the generated output scaffold for generation.
type Output struct {
	Generations []*ActiveGeneration `parameter:"Generations,kind=output,in=view,dataType=[]*ActiveGeneration" view:"generation,type=ActiveGeneration,table=runtime_generations,selectorNoLimit=true" sql:"uri=studio_runtime_generations_store_active_others_generation:sql/read.sql"`
}
