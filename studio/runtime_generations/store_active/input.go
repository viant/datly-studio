package store_active

// Input is the generated input scaffold for definition.
type Input struct {
	NamespaceId string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=false"`
	Has         *InputHas `setMarker:"true" json:"-" sqlx:"-"`
}
type InputHas struct{ NamespaceId bool }
