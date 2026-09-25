package reader

// NamespaceQueryOutput is the generated output scaffold for namespace.
type NamespaceQueryOutput struct {
	Items      []*NamespaceRecord `parameter:"Items,kind=output,in=view,dataType=[]*NamespaceRecord" view:"namespace,type=NamespaceRecord,table=namespaces,limit=500,selectorLimit=true,selectorOffset=true" sql:"uri=studio_namespaces_reader_namespace:sql/namespace.sql"`
	PageLimit  int                `parameter:"PageLimit,kind=output,in=body,dataType=int" json:"limit"`
	PageOffset int                `parameter:"PageOffset,kind=output,in=body,dataType=int" json:"offset"`
}
