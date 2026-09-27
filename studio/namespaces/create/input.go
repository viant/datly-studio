package create

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// NamespaceCreateInput is the generated input scaffold for namespace.
type NamespaceCreateInput struct {
	Jwt       *jwt.Claims              `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth      *studioauth.Output       `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Namespace *NamespaceRecord         `parameter:"Namespace,kind=body,in=,dataType=*NamespaceRecord,required=true" anonymous:"true" view:"namespace,type=NamespaceRecord,table=namespaces" sql:"uri=studio_namespaces_create_namespace:sql/namespace.sql"`
	Has       *NamespaceCreateInputHas `setMarker:"true" typeName:"NamespaceCreateInputHas" json:"-" sqlx:"-"`
}

type NamespaceCreateInputHas struct {
	Jwt       bool
	Auth      bool
	Namespace bool
}
