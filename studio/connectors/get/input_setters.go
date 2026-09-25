package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *ConnectorGetInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &ConnectorGetInputHas{}
	}
	input.Has.Jwt = true
}

func (input *ConnectorGetInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &ConnectorGetInputHas{}
	}
	input.Has.Auth = true
}

func (input *ConnectorGetInput) SetName(value string) {
	if input == nil {
		return
	}
	input.Name = value
	if input.Has == nil {
		input.Has = &ConnectorGetInputHas{}
	}
	input.Has.Name = true
}
