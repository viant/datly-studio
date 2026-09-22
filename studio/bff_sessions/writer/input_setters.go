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

func (input *Input) SetSessions(value []*SessionRevocation) {
	if input == nil {
		return
	}
	input.Sessions = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Sessions = true
}

func (input *Input) SetSessionKeys(value []SessionKeysRow) {
	if input == nil {
		return
	}
	input.SessionKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SessionKeys = true
}

func (input *Input) SetCurrentSession(value []*CurrentSessionView) {
	if input == nil {
		return
	}
	input.CurrentSession = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentSession = true
}
