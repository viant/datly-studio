package store_snapshot

// Input is the generated input scaffold for file.
type Input struct {
	ReportId     string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,f,report_id"`
	VersionNo    int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true" predicate:"equal,f,version_no"`
	ResourceId   string    `parameter:"ResourceId,kind=query,in=resourceId,dataType=string,required=false" predicate:"equal,f,resource_id"`
	Namespace    string    `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=false" predicate:"equal,f,namespace"`
	ResourcePath string    `parameter:"ResourcePath,kind=query,in=resourcePath,dataType=string,required=false" predicate:"equal,f,resource_path"`
	Has          *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId     bool
	VersionNo    bool
	ResourceId   bool
	Namespace    bool
	ResourcePath bool
}
