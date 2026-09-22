package reader

// Output is the generated output scaffold for folder.
type Output struct {
	Folders []*ReportResourceFolder `parameter:"Folders,kind=output,in=view,dataType=[]*ReportResourceFolder" view:"folder,type=ReportResourceFolder,table=report_resource_folders" sql:"uri=studio_report_resource_folders_reader_folder:sql/folder.sql"`
}
