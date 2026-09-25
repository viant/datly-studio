package store_head

// Input is the generated input scaffold for head.
type Input struct {
	ReportId string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	Has      *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId bool
}
