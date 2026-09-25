package store_read

// Output is the generated output scaffold for namespace.
type Output struct {
	Namespaces []*StoredNamespace `parameter:"Namespaces,kind=output,in=view,dataType=[]*StoredNamespace" view:"namespace,type=StoredNamespace,selectorNoLimit=true" sql:"uri=studio_namespaces_store_read_namespace:sql/read.sql"`
}
