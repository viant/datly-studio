package store_readers

// Output is the generated output scaffold for reader.
type Output struct {
	Readers []*PublishedReader `parameter:"Readers,kind=output,in=view,dataType=[]*PublishedReader" view:"reader,type=PublishedReader,table=report_publications,selectorNoLimit=true" sql:"uri=studio_runtime_generations_store_readers_reader:sql/read.sql"`
}
