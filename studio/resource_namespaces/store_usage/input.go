package store_usage

// Input is the generated input scaffold for usage.
type Input struct {
	NamespaceId     string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=true"`
	Namespace       string    `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=true" predicate:"equal,u,namespace"`
	ExcludeReportId string    `parameter:"ExcludeReportId,kind=query,in=excludeReportId,dataType=string,required=true" predicate:"not_equal,u,report_id"`
	Has             *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NamespaceId     bool
	Namespace       bool
	ExcludeReportId bool
}
