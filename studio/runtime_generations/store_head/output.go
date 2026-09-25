package store_head

// Output is the generated output scaffold for head.
type Output struct {
	Heads []*GenerationHead `parameter:"Heads,kind=output,in=view,dataType=[]*GenerationHead" view:"head,type=GenerationHead,table=runtime_generations,limit=1" sql:"uri=studio_runtime_generations_store_head_head:sql/read.sql"`
}
