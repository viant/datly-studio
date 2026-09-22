package reader

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

func (input *Input) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ReportId = true
}

func (input *Input) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.VersionNo = true
}

func (input *Input) SetNamespace(value string) {
	if input == nil {
		return
	}
	input.Namespace = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Namespace = true
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
