package store_write

// Input is the generated input scaffold for file.
type Input struct {
	Files                   []*StoredFile           `parameter:"Files,kind=body,in=data,dataType=[]*StoredFile" view:"file,type=StoredFile,entityHooks=FileStoreRules,table=report_resource_files" sql:"uri=studio_report_resource_files_store_write_file:sql/read.sql"`
	FileKeys                []FileKeysRow           `parameter:"FileKeys,kind=param,in=Files,cardinality=Many" codec:"structql,'uri=studio_report_resource_files_store_write_file:sql/file_keys.sql'"`
	CurrentFile             []*CurrentFileView      `parameter:"CurrentFile,kind=view,in=CurrentFile,cardinality=Many" view:"CurrentFile,table=report_resource_files" sql:"uri=studio_report_resource_files_store_write_file:sql/current_file.sql"`
	_fileHandlerReadIndexes *FileHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                     *InputHas               `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Files       bool
	FileKeys    bool
	CurrentFile bool
}
