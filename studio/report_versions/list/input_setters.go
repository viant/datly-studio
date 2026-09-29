package list

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	listoptions "github.com/viant/datly-studio/studio/report_versions/listoptions"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *VersionListInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.Jwt = true
}

func (input *VersionListInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.Auth = true
}

func (input *VersionListInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.ReportId = true
}

func (input *VersionListInput) SetInput(value listoptions.Options) {
	if input == nil {
		return
	}
	input.Input = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.Input = true
}

func (input *VersionListInput) SetLimit(value int) {
	if input == nil {
		return
	}
	input.Limit = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.Limit = true
}

func (input *VersionListInput) SetOffset(value int) {
	if input == nil {
		return
	}
	input.Offset = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.Offset = true
}

func (input *VersionListInput) SetNamespaceId(value *string) {
	if input == nil {
		return
	}
	input.NamespaceId = value
	if input.Has == nil {
		input.Has = &VersionListInputHas{}
	}
	input.Has.NamespaceId = true
}
