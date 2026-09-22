package reader

// WarmupRunOutput is the generated output scaffold for warmup_run.
type WarmupRunOutput struct {
	WarmupRuns []*WarmupRun `parameter:"WarmupRuns,kind=output,in=view,dataType=[]*WarmupRun" view:"warmup_run,type=WarmupRun,table=report_warmup_runs,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={requested_at,started_at,completed_at,status,duration_ns,entries}" sql:"uri=studio_report_warmup_runs_reader_warmup_run:sql/read.sql"`
}
