package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for report.
type Input struct {
	Jwt           *jwt.Claims        `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth          *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/reports/catalogpredicate.ReportCatalogRead"`
	Query         string             `parameter:"Query,kind=body,in=query,dataType=string,required=false" json:"query" predicate:"contains,r,namespace" predicate:"contains,r,slug" predicate:"contains,r,title" predicate:"contains,r,description"`
	Namespace     string             `parameter:"Namespace,kind=body,in=namespace,dataType=string,required=false" json:"namespace" predicate:"equal,group=1,r,namespace"`
	Status        string             `parameter:"Status,kind=body,in=status,dataType=string,required=false" json:"status" predicate:"equal,group=1,r,status"`
	OwnerId       string             `parameter:"OwnerId,kind=body,in=ownerId,dataType=string,required=false" json:"ownerId" predicate:"equal,group=1,r,owner_id"`
	ConnectorName string             `parameter:"ConnectorName,kind=body,in=connectorName,dataType=string,required=false" json:"connectorName" predicate:"equal,group=1,r,default_connector_name"`
	Fields        []string           `parameter:"Fields,kind=body,in=fields,dataType=[]string,required=false" json:"fields"`
	OrderBy       string             `parameter:"OrderBy,kind=body,in=orderBy,dataType=string,required=false" json:"orderBy"`
	Limit         int                `parameter:"Limit,kind=body,in=limit,dataType=int,required=false,cacheable=false" json:"limit" querySelector:"view=report"`
	Offset        int                `parameter:"Offset,kind=body,in=offset,dataType=int,required=false,cacheable=false" json:"offset" querySelector:"view=report"`
	Has           *InputHas          `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt           bool
	Auth          bool
	Query         bool
	Namespace     bool
	Status        bool
	OwnerId       bool
	ConnectorName bool
	Fields        bool
	OrderBy       bool
	Limit         bool
	Offset        bool
}
