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

func (input *Input) SetExposures(value []*ReportMCPExposure) {
	if input == nil {
		return
	}
	input.Exposures = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Exposures = true
}

func (input *Input) SetExposureKeys(value []ExposureKeysRow) {
	if input == nil {
		return
	}
	input.ExposureKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ExposureKeys = true
}

func (input *Input) SetCurrentExposure(value []*CurrentExposureView) {
	if input == nil {
		return
	}
	input.CurrentExposure = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentExposure = true
}
