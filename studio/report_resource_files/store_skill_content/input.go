package store_skill_content

// Input is the generated input scaffold for file.
type Input struct {
	ReportId     string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,f,report_id"`
	VersionNo    int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true" predicate:"equal,f,version_no"`
	Namespace    string    `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=true" predicate:"equal,f,namespace"`
	ResourcePath string    `parameter:"ResourcePath,kind=query,in=resourcePath,dataType=string,required=true" predicate:"equal,f,resource_path"`
	Has          *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId     bool
	VersionNo    bool
	Namespace    bool
	ResourcePath bool
}
