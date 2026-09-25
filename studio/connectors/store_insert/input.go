package store_insert

// Input is the generated input scaffold for connector.
type Input struct {
	Connectors []*StoredConnector `parameter:"Connectors,kind=body,in=data,dataType=[]*StoredConnector" view:"connector,type=StoredConnector,entityHooks=ConnectorInsertRules,table=connectors" sql:"uri=studio_connectors_store_insert_connector:sql/read.sql"`
	Has        *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Connectors bool
}
