package store_catalog

// Input is the generated input scaffold for generation.
type Input struct {
	NamespaceId  string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=false"`
	GenerationNo int64     `parameter:"GenerationNo,kind=query,in=generationNo,dataType=int64,required=false" predicate:"equal,group=1,g,generation_no"`
	Status       string    `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,g,status"`
	Has          *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NamespaceId  bool
	GenerationNo bool
	Status       bool
}
