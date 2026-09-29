package store_active_others

// Input is the generated input scaffold for publication.
type Input struct {
	NamespaceId     string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=false"`
	ExcludeReportId string    `parameter:"ExcludeReportId,kind=query,in=excludeReportId,dataType=string,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/report_publications/otheractivepredicate.OtherActive"`
	Has             *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NamespaceId     bool
	ExcludeReportId bool
}
