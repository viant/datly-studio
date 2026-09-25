package store_import

// Output is the generated output scaffold for version.
type Output struct {
	Data []*ImportedVersion `parameter:"Data,kind=output,in=body,dataType=[]*ImportedVersion"`
}
