package store_run_access

// Input is the generated input scaffold for access.
type Input struct {
	ReportId string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	Subject  string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/reports/catalogpredicate.RunNamespaceRead"`
	Has      *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId bool
	Subject  bool
}

func (input Input) NamespaceRunSubject() string { return input.Subject }
