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

func (input *Input) SetParameters(value []*ReportParameter) {
	if input == nil {
		return
	}
	input.Parameters = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Parameters = true
}

func (input *Input) SetParameterKeys(value []ParameterKeysRow) {
	if input == nil {
		return
	}
	input.ParameterKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ParameterKeys = true
}

func (input *Input) SetCurrentParameter(value []*CurrentParameterView) {
	if input == nil {
		return
	}
	input.CurrentParameter = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentParameter = true
}

func (input *Input) SetCurrentPredicate(value []*CurrentPredicateView) {
	if input == nil {
		return
	}
	input.CurrentPredicate = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentPredicate = true
}
