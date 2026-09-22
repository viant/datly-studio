package reader

import jwt "github.com/viant/scy/auth/jwt"

// Input is the generated input scaffold for the immutable publication event
// reader. The event predicate restricts the result to the JWT owner's tenant.
type Input struct {
	Jwt       *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportID  string      `parameter:"ReportID,kind=query,in=reportId,dataType=string,required=true" predicate:"equal,group=1,event,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.PublicationEventRead"`
	Operation string      `parameter:"Operation,kind=query,in=operation,dataType=string,required=false" predicate:"equal,group=1,event,operation"`
	Status    string      `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,event,status"`
	Fields    []string    `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=event"`
	OrderBy   string      `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=event"`
	Limit     int         `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=event"`
	Offset    int         `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=event"`
	Has       *InputHas   `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt, ReportID, Operation, Status, Fields, OrderBy, Limit, Offset bool
}
