package store_preview_definition

// Input is the generated input scaffold for definition.
type Input struct {
	ReportId  string    `parameter:"ReportId,kind=query,in=reportId,dataType=string,required=true"`
	VersionNo int       `parameter:"VersionNo,kind=query,in=versionNo,dataType=int,required=true"`
	Has       *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ReportId  bool
	VersionNo bool
}
