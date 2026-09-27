package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *WarmupGetInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &WarmupGetInputHas{}
	}
	input.Has.Jwt = true
}

func (input *WarmupGetInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &WarmupGetInputHas{}
	}
	input.Has.Auth = true
}

func (input *WarmupGetInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &WarmupGetInputHas{}
	}
	input.Has.ReportId = true
}

func (input *WarmupGetInput) SetRunId(value string) {
	if input == nil {
		return
	}
	input.RunId = value
	if input.Has == nil {
		input.Has = &WarmupGetInputHas{}
	}
	input.Has.RunId = true
}
