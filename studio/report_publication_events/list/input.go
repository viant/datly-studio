package list

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	listoptions "github.com/viant/datly-studio/studio/report_publication_events/listoptions"
	jwt "github.com/viant/scy/auth/jwt"
)

// PublicationEventsListInput is the generated input scaffold for event.
type PublicationEventsListInput struct {
	Jwt      *jwt.Claims                    `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Auth     *studioauth.Output             `parameter:"Auth,kind=component,in=GET:/v1/studio/auth/context,dataType=*studioauth.Output,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/authorization.PublicationEventRead"`
	ReportId string                         `parameter:"ReportId,kind=body,in=reportId,dataType=string,required=true" json:"reportId"`
	Input    listoptions.Options            `parameter:"Input,kind=body,in=input,dataType=listoptions.Options,required=false" json:"input"`
	Has      *PublicationEventsListInputHas `setMarker:"true" typeName:"PublicationEventsListInputHas" json:"-" sqlx:"-"`
}

type PublicationEventsListInputHas struct {
	Jwt      bool
	Auth     bool
	ReportId bool
	Input    bool
}
