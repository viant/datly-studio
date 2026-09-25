package store_write

// Input is the generated input scaffold for warmup_run.
type Input struct {
	Runs                         []*StoredWarmupRun           `parameter:"Runs,kind=body,in=data,dataType=[]*StoredWarmupRun" view:"warmup_run,type=StoredWarmupRun,entityHooks=WarmupRunRules,table=report_warmup_runs,selectorNoLimit=true" sql:"uri=studio_report_warmup_runs_store_write_warmup_run:sql/read.sql"`
	WarmupRunKeys                []WarmupRunKeysRow           `parameter:"WarmupRunKeys,kind=param,in=Runs,cardinality=Many" codec:"structql,'uri=studio_report_warmup_runs_store_write_warmup_run:sql/warmup_run_keys.sql'"`
	CurrentWarmupRun             []*CurrentWarmupRunView      `parameter:"CurrentWarmupRun,kind=view,in=CurrentWarmupRun,cardinality=Many" view:"CurrentWarmupRun,table=report_warmup_runs,selectorNoLimit=true" sql:"uri=studio_report_warmup_runs_store_write_warmup_run:sql/current_warmup_run.sql"`
	_warmupRunHandlerReadIndexes *WarmupRunHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                          *InputHas                    `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Runs             bool
	WarmupRunKeys    bool
	CurrentWarmupRun bool
}
