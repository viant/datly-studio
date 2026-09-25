package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for connector.
type Input struct {
	Jwt     *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth    *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ConnectorRead"`
	Query   string             `parameter:"Query,kind=body,in=query,dataType=string,required=false" json:"query" predicate:"contains,c,name" predicate:"contains,c,description" predicate:"contains,c,driver" predicate:"contains,c,owner_id"`
	Status  string             `parameter:"Status,kind=body,in=status,dataType=string,required=false" json:"status" predicate:"equal,group=1,c,status"`
	OwnerID string             `parameter:"OwnerID,kind=body,in=ownerId,dataType=string,required=false" json:"ownerId" predicate:"equal,group=1,c,owner_id"`
	Driver  string             `parameter:"Driver,kind=body,in=driver,dataType=string,required=false" json:"driver" predicate:"equal,group=1,c,driver"`
	Fields  []string           `parameter:"Fields,kind=body,in=fields,dataType=[]string,required=false" json:"fields"`
	OrderBy string             `parameter:"OrderBy,kind=body,in=orderBy,dataType=string,required=false" json:"orderBy"`
	Limit   int                `parameter:"Limit,kind=body,in=limit,dataType=int,required=false,cacheable=false" json:"limit" querySelector:"view=connector"`
	Offset  int                `parameter:"Offset,kind=body,in=offset,dataType=int,required=false,cacheable=false" json:"offset" querySelector:"view=connector"`
	Has     *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt     bool
	Auth    bool
	Query   bool
	Status  bool
	OwnerID bool
	Driver  bool
	Fields  bool
	OrderBy bool
	Limit   bool
	Offset  bool
}
