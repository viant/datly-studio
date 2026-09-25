package store_insert

// Output is the generated output scaffold for event.
type Output struct {
	Data []*StoredEvent `parameter:"Data,kind=output,in=body,dataType=[]*StoredEvent"`
}
