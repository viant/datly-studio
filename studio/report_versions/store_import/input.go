package store_import

// Input is the generated input scaffold for version.
type Input struct {
	Versions []*ImportedVersion `parameter:"Versions,kind=body,in=data,dataType=[]*ImportedVersion" view:"version,type=ImportedVersion,entityHooks=ImportRules,table=report_versions" sql:"uri=studio_report_versions_store_import_version:sql/read.sql"`
	Has      *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Versions bool
}
