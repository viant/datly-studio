package store_download

// Output is the generated output scaffold for file.
type Output struct {
	Files []*DownloadResourceFile `parameter:"Files,kind=output,in=view,dataType=[]*DownloadResourceFile" view:"file,type=DownloadResourceFile,selectorNoLimit=true" sql:"uri=studio_report_resource_files_store_download_file:sql/read.sql"`
}
