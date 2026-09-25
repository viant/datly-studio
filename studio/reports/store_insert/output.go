package store_insert

// Output is the generated output scaffold for report.
type Output struct {
	Data []*StoredReport `parameter:"Data,kind=output,in=body,dataType=[]*StoredReport"`
}
