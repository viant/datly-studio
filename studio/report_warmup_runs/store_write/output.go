package store_write

// Output is the generated output scaffold for warmup_run.
type Output struct {
	Data []*StoredWarmupRun `parameter:"Data,kind=output,in=body,dataType=[]*StoredWarmupRun"`
}
