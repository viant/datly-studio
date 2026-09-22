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

func (input *Input) SetReports(value []*Report) {
	if input == nil {
		return
	}
	input.Reports = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Reports = true
}

func (input *Input) SetReportKeys(value []ReportKeysRow) {
	if input == nil {
		return
	}
	input.ReportKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ReportKeys = true
}

func (input *Input) SetCurrentReport(value []*CurrentReportView) {
	if input == nil {
		return
	}
	input.CurrentReport = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentReport = true
}
