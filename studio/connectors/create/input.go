package create

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// ConnectorCreateInput is the generated input scaffold for connector.
type ConnectorCreateInput struct {
	Jwt       *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Connector *ConnectorRecord         `parameter:"Connector,kind=body,in=,dataType=*ConnectorRecord,required=true" anonymous:"true" view:"connector,type=ConnectorRecord,table=connectors" sql:"uri=studio_connectors_create_connector:sql/connector.sql"`
	Has       *ConnectorCreateInputHas `setMarker:"true" typeName:"ConnectorCreateInputHas" json:"-" sqlx:"-"`
}

type ConnectorCreateInputHas struct {
	Jwt       bool
	Auth      bool
	Connector bool
}
