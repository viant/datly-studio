package store_catalog

// Output is the generated output scaffold for generation.
type Output struct {
	Generations []*CatalogGeneration `parameter:"Generations,kind=output,in=view,dataType=[]*CatalogGeneration" view:"generation,type=CatalogGeneration,table=runtime_generations,selectorNoLimit=true" sql:"uri=studio_runtime_generations_store_catalog_generation:sql/read.sql"`
}
