package store_activate

// Output is the generated output scaffold for publication.
type Output struct {
	Data []*StoredPublication `parameter:"Data,kind=output,in=body,dataType=[]*StoredPublication"`
}
