package create

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *ConnectorCreateInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &ConnectorCreateInputHas{}
	}
	input.Has.Jwt = true
}

func (input *ConnectorCreateInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &ConnectorCreateInputHas{}
	}
	input.Has.Auth = true
}

func (input *ConnectorCreateInput) SetConnector(value *ConnectorRecord) {
	if input == nil {
		return
	}
	input.Connector = value
	if input.Has == nil {
		input.Has = &ConnectorCreateInputHas{}
	}
	input.Has.Connector = true
}
