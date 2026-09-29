package descriptor

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *VersionDescriptorInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &VersionDescriptorInputHas{}
	}
	input.Has.Jwt = true
}

func (input *VersionDescriptorInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &VersionDescriptorInputHas{}
	}
	input.Has.Auth = true
}

func (input *VersionDescriptorInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &VersionDescriptorInputHas{}
	}
	input.Has.ReportId = true
}

func (input *VersionDescriptorInput) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &VersionDescriptorInputHas{}
	}
	input.Has.VersionNo = true
}

func (input *VersionDescriptorInput) SetNamespaceId(value *string) {
	if input == nil {
		return
	}
	input.NamespaceId = value
	if input.Has == nil {
		input.Has = &VersionDescriptorInputHas{}
	}
	input.Has.NamespaceId = true
}
