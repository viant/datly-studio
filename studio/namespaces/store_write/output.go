package store_write

// Output is the generated output scaffold for namespace.
type Output struct {
	Data []*StoredNamespace `parameter:"Data,kind=output,in=body,dataType=[]*StoredNamespace"`
}
