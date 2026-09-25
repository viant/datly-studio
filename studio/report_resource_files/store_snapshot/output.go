package store_snapshot

// Output is the generated output scaffold for file.
type Output struct {
	Files []*SnapshotFile `parameter:"Files,kind=output,in=view,dataType=[]*SnapshotFile" view:"file,type=SnapshotFile,table=report_resource_files,selectorNoLimit=true" sql:"uri=studio_report_resource_files_store_snapshot_file:sql/read.sql"`
}
