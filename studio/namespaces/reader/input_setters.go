package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *NamespaceQueryInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &NamespaceQueryInputHas{}
	}
	input.Has.Jwt = true
}

func (input *NamespaceQueryInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &NamespaceQueryInputHas{}
	}
	input.Has.Auth = true
}

func (input *NamespaceQueryInput) SetQuery(value string) {
	if input == nil {
		return
	}
	input.Query = value
	if input.Has == nil {
		input.Has = &NamespaceQueryInputHas{}
	}
	input.Has.Query = true
}

func (input *NamespaceQueryInput) SetStatus(value string) {
	if input == nil {
		return
	}
	input.Status = value
	if input.Has == nil {
		input.Has = &NamespaceQueryInputHas{}
	}
	input.Has.Status = true
}

func (input *NamespaceQueryInput) SetLimit(value int) {
	if input == nil {
		return
	}
	input.Limit = value
	if input.Has == nil {
		input.Has = &NamespaceQueryInputHas{}
	}
	input.Has.Limit = true
}

func (input *NamespaceQueryInput) SetOffset(value int) {
	if input == nil {
		return
	}
	input.Offset = value
	if input.Has == nil {
		input.Has = &NamespaceQueryInputHas{}
	}
	input.Has.Offset = true
}
