package store_insert

// Input is the generated input scaffold for version.
type Input struct {
	Versions []*StoredVersion `parameter:"Versions,kind=body,in=data,dataType=[]*StoredVersion" view:"version,type=StoredVersion,entityHooks=VersionInsertRules,table=report_versions" sql:"uri=studio_report_versions_store_insert_version:sql/read.sql"`
	Has      *InputHas        `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Versions bool
}
