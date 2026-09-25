package store_write

// Output is the generated output scaffold for folder.
type Output struct {
	Data []*StoredFolder `parameter:"Data,kind=output,in=body,dataType=[]*StoredFolder"`
}
