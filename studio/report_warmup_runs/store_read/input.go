package store_read

// Input is the generated input scaffold for warmup_run.
type Input struct {
	ReportId   string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	RunId      string    `parameter:"RunId,kind=query,in=runId,dataType=string,required=true"`
	ActiveKey  string    `parameter:"ActiveKey,kind=query,in=activeKey,dataType=string,required=true"`
	VersionNo  int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true"`
	PageLimit  int       `parameter:"PageLimit,kind=query,in=pageLimit,dataType=int,required=true"`
	PageOffset int       `parameter:"PageOffset,kind=query,in=pageOffset,dataType=int,required=true"`
	Has        *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId   bool
	RunId      bool
	ActiveKey  bool
	VersionNo  bool
	PageLimit  bool
	PageOffset bool
}
