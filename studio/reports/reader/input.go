package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
)

// Input is the generated input scaffold for report.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true"`
	Id string `parameter:"Id,kind=path,in=id,dataType=string,uri=/{id},required=true" predicate:"equal,group=2,r,id"`
	Query string `parameter:"Query,kind=query,in=q,dataType=string,required=false" predicate:"contains,r,slug" predicate:"contains,r,title" predicate:"contains,r,description"`
	Namespace string `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=false" predicate:"equal,group=1,r,namespace"`
	Status string `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,r,status"`
	OwnerId string `parameter:"OwnerId,kind=query,in=owner,dataType=string,required=false" predicate:"equal,group=1,r,owner_id"`
	ConnectorName string `parameter:"ConnectorName,kind=query,in=connector,dataType=string,required=false" predicate:"equal,group=1,r,default_connector_name"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=report"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=report"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=report"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=report"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Auth bool
	Id bool
	Query bool
	Namespace bool
	Status bool
	OwnerId bool
	ConnectorName bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
