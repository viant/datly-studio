package store_owner

// Input is the generated input scaffold for report.
type Input struct {
	ReportId string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	Has      *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId bool
}
