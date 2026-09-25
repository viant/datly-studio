package store_read

// Output is the generated output scaffold for warmup_run.
type Output struct {
	WarmupRuns []*StoredWarmupRun `parameter:"WarmupRuns,kind=output,in=view,dataType=[]*StoredWarmupRun" view:"warmup_run,type=StoredWarmupRun,selectorNoLimit=true" sql:"uri=studio_report_warmup_runs_store_read_warmup_run:sql/read.sql"`
}
