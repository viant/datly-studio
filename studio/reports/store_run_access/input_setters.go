package store_run_access

func (input *Input) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ReportId = true
}

func (input *Input) SetSubject(value string) {
	if input == nil {
		return
	}
	input.Subject = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Subject = true
}
