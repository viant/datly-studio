package list

// VersionListOutput is the generated output scaffold for version.
type VersionListOutput struct {
	Items      []*Version `parameter:"Items,kind=output,in=view,dataType=[]*Version" view:"version,type=Version,limit=500,selectorLimit=true,selectorOffset=true" sql:"uri=studio_report_versions_list_version:sql/version.sql"`
	PageLimit  int        `parameter:"PageLimit,kind=output,in=body,dataType=int" json:"limit"`
	PageOffset int        `parameter:"PageOffset,kind=output,in=body,dataType=int" json:"offset"`
}
