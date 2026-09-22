package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
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

func (input *Input) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Auth = true
}
