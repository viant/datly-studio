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

func (input *Input) SetConfigs(value []*ReportCubeConfig) {
	if input == nil {
		return
	}
	input.Configs = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Configs = true
}

func (input *Input) SetConfigKeys(value []ConfigKeysRow) {
	if input == nil {
		return
	}
	input.ConfigKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ConfigKeys = true
}

func (input *Input) SetCurrentConfig(value []*CurrentConfigView) {
	if input == nil {
		return
	}
	input.CurrentConfig = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentConfig = true
}
