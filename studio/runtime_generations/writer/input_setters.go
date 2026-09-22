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

func (input *Input) SetGenerations(value []*RuntimeGeneration) {
	if input == nil {
		return
	}
	input.Generations = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Generations = true
}

func (input *Input) SetGenerationKeys(value []GenerationKeysRow) {
	if input == nil {
		return
	}
	input.GenerationKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.GenerationKeys = true
}

func (input *Input) SetCurrentGeneration(value []*CurrentGenerationView) {
	if input == nil {
		return
	}
	input.CurrentGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentGeneration = true
}
