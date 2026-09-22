package writer

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *NamespaceMutationInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &NamespaceMutationInputHas{}
	}
	input.Has.Jwt = true
}

func (input *NamespaceMutationInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &NamespaceMutationInputHas{}
	}
	input.Has.Auth = true
}

func (input *NamespaceMutationInput) SetNamespaces(value []*NamespaceMutationRecord) {
	if input == nil {
		return
	}
	input.Namespaces = value
	if input.Has == nil {
		input.Has = &NamespaceMutationInputHas{}
	}
	input.Has.Namespaces = true
}

func (input *NamespaceMutationInput) SetNamespaceKeys(value []NamespaceKeysRow) {
	if input == nil {
		return
	}
	input.NamespaceKeys = value
	if input.Has == nil {
		input.Has = &NamespaceMutationInputHas{}
	}
	input.Has.NamespaceKeys = true
}

func (input *NamespaceMutationInput) SetCurrentNamespace(value []*CurrentNamespaceView) {
	if input == nil {
		return
	}
	input.CurrentNamespace = value
	if input.Has == nil {
		input.Has = &NamespaceMutationInputHas{}
	}
	input.Has.CurrentNamespace = true
}
