package store_list

// Input is the generated input scaffold for event.
type Input struct {
	ReportId   string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	Operation  string    `parameter:"Operation,kind=query,in=operation,dataType=string,required=false"`
	Status     string    `parameter:"Status,kind=query,in=status,dataType=string,required=false"`
	PageLimit  int       `parameter:"PageLimit,kind=query,in=pageLimit,dataType=int,required=true"`
	PageOffset int       `parameter:"PageOffset,kind=query,in=pageOffset,dataType=int,required=true"`
	Has        *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId   bool
	Operation  bool
	Status     bool
	PageLimit  bool
	PageOffset bool
}
