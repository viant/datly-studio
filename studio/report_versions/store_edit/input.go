package store_edit

// Input is the generated input scaffold for version.
type Input struct {
	Versions                   []*StoredVersion           `parameter:"Versions,kind=body,in=data,dataType=[]*StoredVersion" view:"version,type=StoredVersion,entityHooks=VersionEditRules,table=report_versions" sql:"uri=studio_report_versions_store_edit_version:sql/read.sql"`
	VersionKeys                []VersionKeysRow           `parameter:"VersionKeys,kind=param,in=Versions,cardinality=Many" codec:"structql,'uri=studio_report_versions_store_edit_version:sql/version_keys.sql'"`
	CurrentVersion             []*CurrentVersionView      `parameter:"CurrentVersion,kind=view,in=CurrentVersion,cardinality=Many" view:"CurrentVersion,table=report_versions" sql:"uri=studio_report_versions_store_edit_version:sql/current_version.sql"`
	_versionHandlerReadIndexes *VersionHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                        *InputHas                  `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Versions       bool
	VersionKeys    bool
	CurrentVersion bool
}
