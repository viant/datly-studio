package store_download_budget

// Output is the generated output scaffold for budget.
type Output struct {
	Summary *Budget `parameter:"Summary,kind=output,in=view,dataType=*Budget" view:"budget,type=Budget,table=component_resource_files,limit=1" sql:"uri=studio_report_resource_files_store_download_budget_budget:sql/budget.sql"`
}
