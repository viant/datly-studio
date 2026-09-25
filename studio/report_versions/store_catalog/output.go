package store_catalog

// Output is the generated output scaffold for version.
type Output struct {
	Versions []*StoredVersion `parameter:"Versions,kind=output,in=view,dataType=[]*StoredVersion" view:"version,type=StoredVersion,table=report_versions,limit=500,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={version_no}" sql:"uri=studio_report_versions_store_catalog_version:sql/read.sql"`
}
