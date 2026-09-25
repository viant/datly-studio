package store_catalog

// Input is the generated input scaffold for version.
type Input struct {
	ReportId      string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,group=2,v,report_id"`
	VersionNo     int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=false" predicate:"equal,group=2,v,version_no"`
	State         string    `parameter:"State,kind=query,in=state,dataType=string,required=false" predicate:"equal,group=1,v,state"`
	AuthoringMode string    `parameter:"AuthoringMode,kind=query,in=authoringMode,dataType=string,required=false" predicate:"equal,group=1,v,authoring_mode"`
	CompileStatus string    `parameter:"CompileStatus,kind=query,in=compileStatus,dataType=string,required=false" predicate:"equal,group=1,v,compile_status"`
	CreatedBy     string    `parameter:"CreatedBy,kind=query,in=createdBy,dataType=string,required=false" predicate:"equal,group=1,v,created_by"`
	Limit         int       `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=version"`
	Offset        int       `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=version"`
	Has           *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId      bool
	VersionNo     bool
	State         bool
	AuthoringMode bool
	CompileStatus bool
	CreatedBy     bool
	Limit         bool
	Offset        bool
}
