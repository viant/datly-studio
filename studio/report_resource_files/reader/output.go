package reader

// Output is the generated output scaffold for file.
type Output struct {
	Files []*ReportResourceFile `parameter:"Files,kind=output,in=view,dataType=[]*ReportResourceFile" view:"file,type=ReportResourceFile,table=report_resource_files,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={namespace,resource_path,content_size,created_at}" sql:"uri=studio_report_resource_files_reader_file:sql/file.sql"`
}
