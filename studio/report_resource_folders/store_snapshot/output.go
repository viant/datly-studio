package store_snapshot

// Output is the generated output scaffold for folder.
type Output struct {
	Folders []*SnapshotFolder `parameter:"Folders,kind=output,in=view,dataType=[]*SnapshotFolder" view:"folder,type=SnapshotFolder,table=report_resource_folders,selectorNoLimit=true" sql:"uri=studio_report_resource_folders_store_snapshot_folder:sql/read.sql"`
}
