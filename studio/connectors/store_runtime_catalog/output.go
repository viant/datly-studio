package store_runtime_catalog

// Output is the generated output scaffold for connector.
type Output struct {
	Connectors []*StoredConnector `parameter:"Connectors,kind=output,in=view,dataType=[]*StoredConnector" view:"connector,type=StoredConnector,table=connectors,selectorNoLimit=true" sql:"uri=studio_connectors_store_runtime_catalog_connector:sql/read.sql"`
}
