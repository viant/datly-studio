package store_write

// Input is the generated input scaffold for folder.
type Input struct {
	Folders                   []*StoredFolder           `parameter:"Folders,kind=body,in=data,dataType=[]*StoredFolder" view:"folder,type=StoredFolder,entityHooks=FolderStoreRules,table=report_resource_folders" sql:"uri=studio_report_resource_folders_store_write_folder:sql/read.sql"`
	FolderKeys                []FolderKeysRow           `parameter:"FolderKeys,kind=param,in=Folders,cardinality=Many" codec:"structql,'uri=studio_report_resource_folders_store_write_folder:sql/folder_keys.sql'"`
	CurrentFolder             []*CurrentFolderView      `parameter:"CurrentFolder,kind=view,in=CurrentFolder,cardinality=Many" view:"CurrentFolder,table=report_resource_folders" sql:"uri=studio_report_resource_folders_store_write_folder:sql/current_folder.sql"`
	_folderHandlerReadIndexes *FolderHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                       *InputHas                 `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Folders       bool
	FolderKeys    bool
	CurrentFolder bool
}
