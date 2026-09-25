package store_one

// Input is the generated input scaffold for acl.
type Input struct {
	ReportId    string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	SubjectType string    `parameter:"SubjectType,kind=query,in=subjectType,dataType=string,required=true"`
	SubjectId   string    `parameter:"SubjectId,kind=query,in=subjectId,dataType=string,required=true"`
	Has         *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId    bool
	SubjectType bool
	SubjectId   bool
}
