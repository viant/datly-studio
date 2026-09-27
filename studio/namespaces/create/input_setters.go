package create

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *NamespaceCreateInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &NamespaceCreateInputHas{}
	}
	input.Has.Jwt = true
}

func (input *NamespaceCreateInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &NamespaceCreateInputHas{}
	}
	input.Has.Auth = true
}

func (input *NamespaceCreateInput) SetNamespace(value *NamespaceRecord) {
	if input == nil {
		return
	}
	input.Namespace = value
	if input.Has == nil {
		input.Has = &NamespaceCreateInputHas{}
	}
	input.Has.Namespace = true
}
