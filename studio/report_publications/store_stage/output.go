package store_stage

// Output is the generated output scaffold for publication.
type Output struct {
	Data []*StoredPublication `parameter:"Data,kind=output,in=body,dataType=[]*StoredPublication"`
}
