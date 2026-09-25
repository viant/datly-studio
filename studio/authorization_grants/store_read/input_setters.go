package store_read

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

func (input *Input) SetGrantSource(value string) {
	if input == nil {
		return
	}
	input.GrantSource = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.GrantSource = true
}

func (input *Input) SetCanPublish(value bool) {
	if input == nil {
		return
	}
	input.CanPublish = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CanPublish = true
}

func (input *Input) SetIsLive(value bool) {
	if input == nil {
		return
	}
	input.IsLive = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.IsLive = true
}
