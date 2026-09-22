package reader

// NamespaceQueryOutput is the generated output scaffold for namespace.
type NamespaceQueryOutput struct {
	Namespaces []*NamespaceRecord `parameter:"Namespaces,kind=output,in=view,dataType=[]*NamespaceRecord" view:"namespace,type=NamespaceRecord,table=namespaces,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={name,title,status,updated_at}" sql:"uri=studio_namespaces_reader_namespace:sql/namespace.sql"`
}
