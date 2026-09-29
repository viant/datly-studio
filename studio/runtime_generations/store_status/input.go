package store_status

// Input is the generated input scaffold for generation.
type Input struct {
	NamespaceId string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=false"`
	Status      string    `parameter:"Status,kind=query,in=status,dataType=string,required=true" predicate:"equal,group=1,g,status"`
	Has         *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NamespaceId bool
	Status      bool
}
