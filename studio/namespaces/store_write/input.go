package store_write

// Input is the generated input scaffold for namespace.
type Input struct {
	Namespaces                   []*StoredNamespace           `parameter:"Namespaces,kind=body,in=data,dataType=[]*StoredNamespace" view:"namespace,type=StoredNamespace,entityHooks=NamespaceStoreRules,table=namespaces" sql:"uri=studio_namespaces_store_write_namespace:sql/read.sql"`
	NamespaceKeys                []NamespaceKeysRow           `parameter:"NamespaceKeys,kind=param,in=Namespaces,cardinality=Many" codec:"structql,'uri=studio_namespaces_store_write_namespace:sql/namespace_keys.sql'"`
	CurrentNamespace             []*CurrentNamespaceView      `parameter:"CurrentNamespace,kind=view,in=CurrentNamespace,cardinality=Many" view:"CurrentNamespace,table=namespaces" sql:"uri=studio_namespaces_store_write_namespace:sql/current_namespace.sql"`
	_namespaceHandlerReadIndexes *NamespaceHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                          *InputHas                    `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Namespaces       bool
	NamespaceKeys    bool
	CurrentNamespace bool
}
