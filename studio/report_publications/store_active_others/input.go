package store_active_others

// Input is the generated input scaffold for publication.
type Input struct {
	ExcludeReportId string    `parameter:"ExcludeReportId,kind=query,in=excludeReportId,dataType=string,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/report_publications/otheractivepredicate.OtherActive"`
	Has             *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ExcludeReportId bool
}
