package store_validation

// Output is the generated output scaffold for version.
type Output struct {
	Data []*StoredVersion `parameter:"Data,kind=output,in=body,dataType=[]*StoredVersion"`
}
