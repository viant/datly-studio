package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for connector.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Connectors []*Connector `parameter:"Connectors,kind=body,in=data,dataType=[]*Connector" view:"connector,type=Connector,table=connectors" sql:"uri=studio_connectors_writer_connector:sql/connector.sql"`
	ConnectorKeys []ConnectorKeysRow `parameter:"ConnectorKeys,kind=param,in=Connectors,cardinality=Many" codec:"structql,'uri=studio_connectors_writer_connector:sql/connector_keys.sql'"`
	CurrentConnector []*CurrentConnectorView `parameter:"CurrentConnector,kind=view,in=CurrentConnector,cardinality=Many" view:"CurrentConnector,table=connectors" sql:"uri=studio_connectors_writer_connector:sql/current_connector.sql"`
	_connectorHandlerReadIndexes *ConnectorHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Connectors bool
	ConnectorKeys bool
	CurrentConnector bool
}
