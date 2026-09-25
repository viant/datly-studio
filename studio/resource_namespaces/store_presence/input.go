package store_presence

// Input is the generated input scaffold for usage.
type Input struct {
	Namespace string    `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=true" predicate:"equal,group=1,u,namespace"`
	ReportId  string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,group=1,u,report_id"`
	Has       *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Namespace bool
	ReportId  bool
}
