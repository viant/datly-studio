package store_expired

// Output is the generated output scaffold for warmup_run.
type Output struct {
	WarmupRuns []*ExpiredWarmupRun `parameter:"WarmupRuns,kind=output,in=view,dataType=[]*ExpiredWarmupRun" view:"warmup_run,type=ExpiredWarmupRun,selectorNoLimit=true" sql:"uri=studio_report_warmup_runs_store_expired_warmup_run:sql/read.sql"`
}
