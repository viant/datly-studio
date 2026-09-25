package store_catalog

// Output is the generated output scaffold for connector.
type Output struct {
	Connectors []*StoredConnector `parameter:"Connectors,kind=output,in=view,dataType=[]*StoredConnector" view:"connector,type=StoredConnector,selectorNoLimit=true" sql:"uri=studio_connectors_store_catalog_connector:sql/read.sql"`
}
