package store_active

// Output is the generated output scaffold for connector.
type Output struct {
	Connectors []*StoredConnector `parameter:"Connectors,kind=output,in=view,dataType=[]*StoredConnector" view:"connector,type=StoredConnector,table=connectors,selectorNoLimit=true" sql:"uri=studio_connectors_store_active_connector:sql/read.sql"`
}
