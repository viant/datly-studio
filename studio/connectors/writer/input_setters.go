package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *Input) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Jwt = true
}

func (input *Input) SetConnectors(value []*Connector) {
	if input == nil {
		return
	}
	input.Connectors = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Connectors = true
}

func (input *Input) SetConnectorKeys(value []ConnectorKeysRow) {
	if input == nil {
		return
	}
	input.ConnectorKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ConnectorKeys = true
}

func (input *Input) SetCurrentConnector(value []*CurrentConnectorView) {
	if input == nil {
		return
	}
	input.CurrentConnector = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentConnector = true
}
