package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for connector.
type Input struct {
	Jwt     *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ConnectorRead"`
	Name    string             `parameter:"Name,kind=path,in=name,dataType=string,uri=/{name},required=true" predicate:"equal,group=2,c,name"`
	Query   string             `parameter:"Query,kind=query,in=q,dataType=string,required=false" predicate:"contains,c,name" predicate:"contains,c,description"`
	Status  string             `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,c,status"`
	OwnerID string             `parameter:"OwnerID,kind=query,in=owner,dataType=string,required=false" predicate:"equal,group=1,c,owner_id"`
	Driver  string             `parameter:"Driver,kind=query,in=driver,dataType=string,required=false" predicate:"equal,group=1,c,driver"`
	Fields  []string           `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=connector"`
	OrderBy string             `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=connector"`
	Limit   int                `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=connector"`
	Offset  int                `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=connector"`
	Has     *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt     bool
	Auth    bool
	Name    bool
	Query   bool
	Status  bool
	OwnerID bool
	Driver  bool
	Fields  bool
	OrderBy bool
	Limit   bool
	Offset  bool
}
