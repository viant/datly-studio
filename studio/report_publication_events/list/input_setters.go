package list

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	listoptions "github.com/viant/datly-studio/studio/report_publication_events/listoptions"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *PublicationEventsListInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &PublicationEventsListInputHas{}
	}
	input.Has.Jwt = true
}

func (input *PublicationEventsListInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &PublicationEventsListInputHas{}
	}
	input.Has.Auth = true
}

func (input *PublicationEventsListInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &PublicationEventsListInputHas{}
	}
	input.Has.ReportId = true
}

func (input *PublicationEventsListInput) SetInput(value listoptions.Options) {
	if input == nil {
		return
	}
	input.Input = value
	if input.Has == nil {
		input.Has = &PublicationEventsListInputHas{}
	}
	input.Has.Input = true
}
