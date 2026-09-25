package get

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *NamespaceGetInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &NamespaceGetInputHas{}
	}
	input.Has.Jwt = true
}

func (input *NamespaceGetInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &NamespaceGetInputHas{}
	}
	input.Has.Auth = true
}

func (input *NamespaceGetInput) SetName(value string) {
	if input == nil {
		return
	}
	input.Name = value
	if input.Has == nil {
		input.Has = &NamespaceGetInputHas{}
	}
	input.Has.Name = true
}
