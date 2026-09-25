package store_snapshot

// Input is the generated input scaffold for folder.
type Input struct {
	ReportId  string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,f,report_id"`
	VersionNo int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true" predicate:"equal,f,version_no"`
	FolderId  string    `parameter:"FolderId,kind=query,in=folderId,dataType=string,required=false" predicate:"equal,f,folder_id"`
	Has       *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId  bool
	VersionNo bool
	FolderId  bool
}
