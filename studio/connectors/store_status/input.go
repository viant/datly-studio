package store_status

// Input is the generated input scaffold for connector.
type Input struct {
	Operation                    string                       `parameter:"Operation,kind=query,in=operation,dataType=string,required=true"`
	Connectors                   []*StoredConnector           `parameter:"Connectors,kind=body,in=data,dataType=[]*StoredConnector" view:"connector,type=StoredConnector,entityHooks=ConnectorStatusRules,table=connectors" sql:"uri=studio_connectors_store_status_connector:sql/read.sql"`
	ConnectorKeys                []ConnectorKeysRow           `parameter:"ConnectorKeys,kind=param,in=Connectors,cardinality=Many" codec:"structql,'uri=studio_connectors_store_status_connector:sql/connector_keys.sql'"`
	CurrentConnector             []*CurrentConnectorView      `parameter:"CurrentConnector,kind=view,in=CurrentConnector,cardinality=Many" view:"CurrentConnector,table=connectors" sql:"uri=studio_connectors_store_status_connector:sql/current_connector.sql"`
	_connectorHandlerReadIndexes *ConnectorHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                          *InputHas                    `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Operation        bool
	Connectors       bool
	ConnectorKeys    bool
	CurrentConnector bool
}
