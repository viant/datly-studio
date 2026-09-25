package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for acl.
type Input struct {
	Jwt         *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	ReportId    string      `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId" predicate:"equal,group=2,a,report_id" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.ACLRead"`
	SubjectType string      `parameter:"SubjectType,kind=query,in=subjectType,dataType=string,required=false" json:"subjectType" predicate:"equal,group=1,a,subject_type"`
	SubjectId   string      `parameter:"SubjectId,kind=query,in=subjectId,dataType=string,required=false" json:"subjectId" predicate:"equal,group=1,a,subject_id"`
	Has         *InputHas   `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt         bool
	ReportId    bool
	SubjectType bool
	SubjectId   bool
}
