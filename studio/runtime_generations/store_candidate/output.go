package store_candidate

// Output is the generated output scaffold for definition.
type Output struct {
	Definitions []*PublishedDefinition `parameter:"Definitions,kind=output,in=view,dataType=[]*PublishedDefinition" view:"definition,type=PublishedDefinition,table=report_publications,selectorNoLimit=true" sql:"uri=studio_runtime_generations_store_candidate_definition:sql/read.sql"`
}
