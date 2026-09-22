package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *Input) mark() *InputHas {
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	return input.Has
}
func (input *Input) SetJwt(value *jwt.Claims) {
	if input != nil {
		input.Jwt = value
		input.mark().Jwt = true
	}
}
func (input *Input) SetReportID(value string) {
	if input != nil {
		input.ReportID = value
		input.mark().ReportID = true
	}
}
func (input *Input) SetOperation(value string) {
	if input != nil {
		input.Operation = value
		input.mark().Operation = true
	}
}
func (input *Input) SetStatus(value string) {
	if input != nil {
		input.Status = value
		input.mark().Status = true
	}
}
func (input *Input) SetFields(value []string) {
	if input != nil {
		input.Fields = value
		input.mark().Fields = true
	}
}
func (input *Input) SetOrderBy(value string) {
	if input != nil {
		input.OrderBy = value
		input.mark().OrderBy = true
	}
}
func (input *Input) SetLimit(value int) {
	if input != nil {
		input.Limit = value
		input.mark().Limit = true
	}
}
func (input *Input) SetOffset(value int) {
	if input != nil {
		input.Offset = value
		input.mark().Offset = true
	}
}
