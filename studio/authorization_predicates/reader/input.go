package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// AuthorizationPredicateQueryInput is the generated input scaffold for authorization_predicate.
type AuthorizationPredicateQueryInput struct {
	Jwt     *jwt.Claims                          `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Name    string                               `parameter:"Name,kind=path,in=name,dataType=string,uri=/{name},required=true" predicate:"equal,group=2,p,name" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.AuthorizationPredicateRead"`
	Query   string                               `parameter:"Query,kind=query,in=q,dataType=string,required=false" predicate:"contains,p,name" predicate:"contains,p,title" predicate:"contains,p,package_path" predicate:"contains,p,type_name"`
	Status  string                               `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,p,status"`
	Fields  []string                             `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	OrderBy string                               `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	Limit   int                                  `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	Offset  int                                  `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	Has     *AuthorizationPredicateQueryInputHas `setMarker:"true" typeName:"AuthorizationPredicateQueryInputHas" json:"-" sqlx:"-"`
}

type AuthorizationPredicateQueryInputHas struct {
	Jwt     bool
	Name    bool
	Query   bool
	Status  bool
	Fields  bool
	OrderBy bool
	Limit   bool
	Offset  bool
}
