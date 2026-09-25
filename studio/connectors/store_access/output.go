package store_access

// Output is the generated output scaffold for connector.
type Output struct {
	Connectors []*StoredConnector `parameter:"Connectors,kind=output,in=view,dataType=[]*StoredConnector" view:"connector,type=StoredConnector,table=connectors,limit=1" sql:"uri=studio_connectors_store_access_connector:sql/read.sql"`
}
