package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for file.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Files []*ReportResourceFile `parameter:"Files,kind=body,in=data,dataType=[]*ReportResourceFile" view:"file,type=ReportResourceFile,entityHooks=FileRules,table=report_resource_files" sql:"uri=studio_report_resource_files_writer_file:sql/file.sql"`
	FileKeys []FileKeysRow `parameter:"FileKeys,kind=param,in=Files,cardinality=Many" codec:"structql,'uri=studio_report_resource_files_writer_file:sql/file_keys.sql'"`
	CurrentFile []*CurrentFileView `parameter:"CurrentFile,kind=view,in=CurrentFile,cardinality=Many" view:"CurrentFile,table=report_resource_files" sql:"uri=studio_report_resource_files_writer_file:sql/current_file.sql"`
	_fileHandlerReadIndexes *FileHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Files bool
	FileKeys bool
	CurrentFile bool
}
