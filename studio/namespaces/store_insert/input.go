package store_insert

// Input is the generated input scaffold for namespace.
type Input struct {
	Namespaces []*StoredNamespace `parameter:"Namespaces,kind=body,in=data,dataType=[]*StoredNamespace" view:"namespace,type=StoredNamespace,entityHooks=NamespaceInsertRules,table=namespaces" sql:"uri=studio_namespaces_store_insert_namespace:sql/read.sql"`
	Has        *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Namespaces bool
}
