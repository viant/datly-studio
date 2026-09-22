package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
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

func (input *Input) SetStatus(value string) {
	if input == nil {
		return
	}
	input.Status = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Status = true
}

func (input *Input) SetSourceRevision(value string) {
	if input == nil {
		return
	}
	input.SourceRevision = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SourceRevision = true
}

func (input *Input) SetFields(value []string) {
	if input == nil {
		return
	}
	input.Fields = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Fields = true
}

func (input *Input) SetOrderBy(value string) {
	if input == nil {
		return
	}
	input.OrderBy = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.OrderBy = true
}

func (input *Input) SetLimit(value int) {
	if input == nil {
		return
	}
	input.Limit = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Limit = true
}

func (input *Input) SetOffset(value int) {
	if input == nil {
		return
	}
	input.Offset = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Offset = true
}
