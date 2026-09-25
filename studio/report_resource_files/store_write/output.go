package store_write

// Output is the generated output scaffold for file.
type Output struct {
	Data []*StoredFile `parameter:"Data,kind=output,in=body,dataType=[]*StoredFile"`
}
