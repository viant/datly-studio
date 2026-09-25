package store_access

// Output is the generated output scaffold for namespace.
type Output struct {
	Namespaces []*StoredNamespace `parameter:"Namespaces,kind=output,in=view,dataType=[]*StoredNamespace" view:"namespace,type=StoredNamespace,table=namespaces,limit=1" sql:"uri=studio_namespaces_store_access_namespace:sql/read.sql"`
}
