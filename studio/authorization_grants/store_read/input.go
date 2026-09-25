package store_read

// Input is the generated input scaffold for grant.
type Input struct {
	SubjectId   string    `parameter:"SubjectId,kind=query,in=subjectId,dataType=string,required=true" predicate:"equal,group=1,g,subject_id"`
	SubjectType string    `parameter:"SubjectType,kind=query,in=subjectType,dataType=string,required=true" predicate:"equal,group=1,g,subject_type"`
	ReportId    string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=false" predicate:"equal,group=1,g,report_id"`
	GrantSource string    `parameter:"GrantSource,kind=query,in=grantSource,dataType=string,required=false" predicate:"equal,group=1,g,grant_source"`
	CanPublish  bool      `parameter:"CanPublish,kind=query,in=canPublish,dataType=bool,required=false" predicate:"equal,group=1,g,can_publish"`
	IsLive      bool      `parameter:"IsLive,kind=query,in=isLive,dataType=bool,required=true" predicate:"equal,group=1,g,is_live"`
	Has         *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	SubjectId   bool
	SubjectType bool
	ReportId    bool
	GrantSource bool
	CanPublish  bool
	IsLive      bool
}
