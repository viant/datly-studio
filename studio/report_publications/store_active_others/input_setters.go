package store_active_others

func (input *Input) SetExcludeReportId(value string) {
	if input == nil {
		return
	}
	input.ExcludeReportId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ExcludeReportId = true
}
