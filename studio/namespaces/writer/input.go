package writer

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// NamespaceMutationInput is the generated input scaffold for namespace.
type NamespaceMutationInput struct {
	Jwt                          *jwt.Claims                  `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth                         *studioauth.Output           `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Namespaces                   []*NamespaceMutationRecord   `parameter:"Namespaces,kind=body,in=data,dataType=[]*NamespaceMutationRecord" view:"namespace,type=NamespaceMutationRecord,table=namespaces" sql:"uri=studio_namespaces_writer_namespace:sql/namespace.sql"`
	NamespaceKeys                []NamespaceKeysRow           `parameter:"NamespaceKeys,kind=param,in=Namespaces,cardinality=Many" codec:"structql,'uri=studio_namespaces_writer_namespace:sql/namespace_keys.sql'"`
	CurrentNamespace             []*CurrentNamespaceView      `parameter:"CurrentNamespace,kind=view,in=CurrentNamespace,cardinality=Many" view:"CurrentNamespace,table=namespaces" sql:"uri=studio_namespaces_writer_namespace:sql/current_namespace.sql"`
	_namespaceHandlerReadIndexes *NamespaceHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                          *NamespaceMutationInputHas   `setMarker:"true" typeName:"NamespaceMutationInputHas" json:"-" sqlx:"-"`
}

type NamespaceMutationInputHas struct {
	Jwt              bool
	Auth             bool
	Namespaces       bool
	NamespaceKeys    bool
	CurrentNamespace bool
}
