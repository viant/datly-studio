package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// ConnectorGetInput is the generated input scaffold for connector.
type ConnectorGetInput struct {
	Jwt  *jwt.Claims           `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output    `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ConnectorRead"`
	Name string                `parameter:"Name,kind=body,in=name,dataType=string,required=true" json:"name" predicate:"equal,group=2,c,name"`
	Has  *ConnectorGetInputHas `setMarker:"true" typeName:"ConnectorGetInputHas" json:"-" sqlx:"-"`
}

type ConnectorGetInputHas struct {
	Jwt  bool
	Auth bool
	Name bool
}
