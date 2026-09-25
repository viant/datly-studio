package store_global_access

// Input is the generated input scaffold for report.
type Input struct {
	Subject string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/reports/globalpredicate.GlobalPublish"`
	Has     *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Subject bool
}
