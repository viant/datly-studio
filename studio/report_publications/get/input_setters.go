package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *PublicationGetInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &PublicationGetInputHas{}
	}
	input.Has.Jwt = true
}

func (input *PublicationGetInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &PublicationGetInputHas{}
	}
	input.Has.Auth = true
}

func (input *PublicationGetInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &PublicationGetInputHas{}
	}
	input.Has.ReportId = true
}
