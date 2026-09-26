package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *VersionGetInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &VersionGetInputHas{}
	}
	input.Has.Jwt = true
}

func (input *VersionGetInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &VersionGetInputHas{}
	}
	input.Has.Auth = true
}

func (input *VersionGetInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &VersionGetInputHas{}
	}
	input.Has.ReportId = true
}

func (input *VersionGetInput) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &VersionGetInputHas{}
	}
	input.Has.VersionNo = true
}
