package reader

// Output is the generated output scaffold for generation.
type Output struct {
	Generations []*RuntimeGeneration `parameter:"Generations,kind=output,in=view,dataType=[]*RuntimeGeneration" view:"generation,type=RuntimeGeneration,table=runtime_generations,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={generation_no,status,report_count,requested_at,activated_at,retired_at}" sql:"uri=studio_runtime_generations_reader_generation:sql/generation.sql"`
}
