package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for folder.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Folders []*ReportResourceFolder `parameter:"Folders,kind=body,in=data,dataType=[]*ReportResourceFolder" view:"folder,type=ReportResourceFolder,entityHooks=FolderRules,table=report_resource_folders" sql:"uri=studio_report_resource_folders_writer_folder:sql/folder.sql"`
	FolderKeys []FolderKeysRow `parameter:"FolderKeys,kind=param,in=Folders,cardinality=Many" codec:"structql,'uri=studio_report_resource_folders_writer_folder:sql/folder_keys.sql'"`
	CurrentFolder []*CurrentFolderView `parameter:"CurrentFolder,kind=view,in=CurrentFolder,cardinality=Many" view:"CurrentFolder,table=report_resource_folders" sql:"uri=studio_report_resource_folders_writer_folder:sql/current_folder.sql"`
	_folderHandlerReadIndexes *FolderHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Folders bool
	FolderKeys bool
	CurrentFolder bool
}
