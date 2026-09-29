package export_dql

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *VersionExportInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &VersionExportInputHas{}
	}
	input.Has.Jwt = true
}

func (input *VersionExportInput) SetAuth(value *studioauth.Output) {
	if input == nil {
		return
	}
	input.Auth = value
	if input.Has == nil {
		input.Has = &VersionExportInputHas{}
	}
	input.Has.Auth = true
}

func (input *VersionExportInput) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &VersionExportInputHas{}
	}
	input.Has.ReportId = true
}

func (input *VersionExportInput) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &VersionExportInputHas{}
	}
	input.Has.VersionNo = true
}

func (input *VersionExportInput) SetNamespaceId(value *string) {
	if input == nil {
		return
	}
	input.NamespaceId = value
	if input.Has == nil {
		input.Has = &VersionExportInputHas{}
	}
	input.Has.NamespaceId = true
}
