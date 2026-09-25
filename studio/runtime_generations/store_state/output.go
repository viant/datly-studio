package store_state

// Output is the generated output scaffold for generation.
type Output struct {
	Data []*StoredGeneration `parameter:"Data,kind=output,in=body,dataType=[]*StoredGeneration"`
}
