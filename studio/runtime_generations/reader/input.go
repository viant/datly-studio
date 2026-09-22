package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
)

// Input is the generated input scaffold for generation.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth *studioauth.Output `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.RuntimeRead"`
	Status string `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,g,status"`
	SourceRevision string `parameter:"SourceRevision,kind=query,in=sourceRevision,dataType=string,required=false" predicate:"equal,group=1,g,source_revision"`
	Fields []string `parameter:"Fields,kind=query,in=fields,dataType=[]string,required=false,cacheable=false" querySelector:"view=generation"`
	OrderBy string `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=generation"`
	Limit int `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=generation"`
	Offset int `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=generation"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Auth bool
	Status bool
	SourceRevision bool
	Fields bool
	OrderBy bool
	Limit bool
	Offset bool
}
