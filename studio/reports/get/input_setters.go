package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *ReportGetInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &ReportGetInputHas{}
	}
	input.Has.Jwt = true
}

func (input *ReportGetInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &ReportGetInputHas{}
	}
	input.Has.Auth = true
}

func (input *ReportGetInput) SetId(value string) {
	if input == nil {
		return
	}
	input.Id = value
	if input.Has == nil {
		input.Has = &ReportGetInputHas{}
	}
	input.Has.Id = true
}
