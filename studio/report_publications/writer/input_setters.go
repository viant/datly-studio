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

func (input *Input) SetPublications(value []*ReportPublication) {
	if input == nil {
		return
	}
	input.Publications = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Publications = true
}

func (input *Input) SetPublicationKeys(value []PublicationKeysRow) {
	if input == nil {
		return
	}
	input.PublicationKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.PublicationKeys = true
}

func (input *Input) SetCurrentPublication(value []*CurrentPublicationView) {
	if input == nil {
		return
	}
	input.CurrentPublication = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentPublication = true
}
