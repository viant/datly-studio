package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// NamespaceQueryInput is the generated input scaffold for namespace.
type NamespaceQueryInput struct {
	Jwt     *jwt.Claims             `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output      `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Name    string                  `parameter:"Name,kind=path,in=name,dataType=string,uri=/{name},required=true" predicate:"equal,group=2,n,name"`
	Query   string                  `parameter:"Query,kind=query,in=q,dataType=string,required=false" predicate:"contains,n,name" predicate:"contains,n,title" predicate:"contains,n,description"`
	Status  string                  `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,n,status"`
	Fields  []string                `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=namespace"`
	OrderBy string                  `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=namespace"`
	Limit   int                     `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=namespace"`
	Offset  int                     `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=namespace"`
	Has     *NamespaceQueryInputHas `setMarker:"true" typeName:"NamespaceQueryInputHas" json:"-" sqlx:"-"`
}

type NamespaceQueryInputHas struct {
	Jwt     bool
	Auth    bool
	Name    bool
	Query   bool
	Status  bool
	Fields  bool
	OrderBy bool
	Limit   bool
	Offset  bool
}
