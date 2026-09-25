package reader

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *WarmupRunInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.Jwt = true
}

func (input *WarmupRunInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.Auth = true
}

func (input *WarmupRunInput) SetReportID(value string) {
	if input == nil {
		return
	}
	input.ReportID = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.ReportID = true
}

func (input *WarmupRunInput) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.VersionNo = true
}

func (input *WarmupRunInput) SetRunID(value string) {
	if input == nil {
		return
	}
	input.RunID = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.RunID = true
}

func (input *WarmupRunInput) SetStatus(value string) {
	if input == nil {
		return
	}
	input.Status = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.Status = true
}

func (input *WarmupRunInput) SetFields(value []string) {
	if input == nil {
		return
	}
	input.Fields = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.Fields = true
}

func (input *WarmupRunInput) SetOrderBy(value string) {
	if input == nil {
		return
	}
	input.OrderBy = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.OrderBy = true
}

func (input *WarmupRunInput) SetLimit(value int) {
	if input == nil {
		return
	}
	input.Limit = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.Limit = true
}

func (input *WarmupRunInput) SetOffset(value int) {
	if input == nil {
		return
	}
	input.Offset = value
	if input.Has == nil {
		input.Has = &WarmupRunInputHas{}
	}
	input.Has.Offset = true
}
