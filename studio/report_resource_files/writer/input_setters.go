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

func (input *Input) SetFiles(value []*ReportResourceFile) {
	if input == nil {
		return
	}
	input.Files = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Files = true
}

func (input *Input) SetFileKeys(value []FileKeysRow) {
	if input == nil {
		return
	}
	input.FileKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.FileKeys = true
}

func (input *Input) SetCurrentFile(value []*CurrentFileView) {
	if input == nil {
		return
	}
	input.CurrentFile = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentFile = true
}
