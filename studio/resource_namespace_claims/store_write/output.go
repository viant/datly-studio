package store_write

// Output is the generated output scaffold for claim.
type Output struct {
	Data []*StoredClaim `parameter:"Data,kind=output,in=body,dataType=[]*StoredClaim"`
}
