package store_status

// Output is the generated output scaffold for generation.
type Output struct {
	Generations []*StoredGeneration `parameter:"Generations,kind=output,in=view,dataType=[]*StoredGeneration" view:"generation,type=StoredGeneration,table=runtime_generations,limit=1" sql:"uri=studio_runtime_generations_store_status_generation:sql/read.sql"`
}
