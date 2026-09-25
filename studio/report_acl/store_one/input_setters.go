package store_one

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

func (input *Input) SetSubjectType(value string) {
	if input == nil {
		return
	}
	input.SubjectType = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SubjectType = true
}

func (input *Input) SetSubjectId(value string) {
	if input == nil {
		return
	}
	input.SubjectId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SubjectId = true
}
