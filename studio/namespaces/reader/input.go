package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// NamespaceQueryInput is the generated input scaffold for namespace.
type NamespaceQueryInput struct {
	Jwt    *jwt.Claims             `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth   *studioauth.Output      `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.NamespaceRead"`
	Query  string                  `parameter:"Query,kind=body,in=query,dataType=string,required=false" json:"query" predicate:"contains,n,name" predicate:"contains,n,title" predicate:"contains,n,description"`
	Status string                  `parameter:"Status,kind=body,in=status,dataType=string,required=false" json:"status" predicate:"equal,group=1,n,status"`
	Limit  int                     `parameter:"Limit,kind=body,in=limit,dataType=int,required=false,cacheable=false" json:"limit" querySelector:"view=namespace"`
	Offset int                     `parameter:"Offset,kind=body,in=offset,dataType=int,required=false,cacheable=false" json:"offset" querySelector:"view=namespace"`
	Has    *NamespaceQueryInputHas `setMarker:"true" typeName:"NamespaceQueryInputHas" json:"-" sqlx:"-"`
}

type NamespaceQueryInputHas struct {
	Jwt    bool
	Auth   bool
	Query  bool
	Status bool
	Limit  bool
	Offset bool
}
