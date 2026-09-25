package store_run_access

// Input is the generated input scaffold for access.
type Input struct {
	ReportId string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	Subject  string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true"`
	Has      *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId bool
	Subject  bool
}
