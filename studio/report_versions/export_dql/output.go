package export_dql

// VersionExportOutput is the generated output scaffold for version.
type VersionExportOutput struct {
	Item     *VersionSource `parameter:"Item,kind=output,in=view,dataType=*VersionSource" json:"-" view:"version,type=VersionSource,table=component_versions,limit=1" sql:"uri=studio_report_versions_export_dql_version:sql/version.sql"`
	Dql      string         `parameter:"Dql,kind=output,in=body,dataType=string" json:"dql"`
	Complete bool           `parameter:"Complete,kind=output,in=body,dataType=bool" json:"complete"`
}
