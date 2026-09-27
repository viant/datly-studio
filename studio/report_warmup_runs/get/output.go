package get

import (
	sdk "github.com/viant/datly-studio/sdk"
)

// WarmupGetOutput is the generated output scaffold for warmup_run.
type WarmupGetOutput struct {
	Item     *WarmupRow     `parameter:"Item,kind=output,in=view,dataType=*WarmupRow" json:"-" view:"warmup_run,type=WarmupRow,table=report_warmup_runs,limit=1" sql:"uri=studio_report_warmup_runs_get_warmup_run:sql/warmup_run.sql"`
	Response *sdk.WarmupRun `parameter:"Response,kind=output,in=body,dataType=*sdk.WarmupRun" json:"-"`
}
