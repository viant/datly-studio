package store_skill_content

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

func (input *Input) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.VersionNo = true
}

func (input *Input) SetNamespace(value string) {
	if input == nil {
		return
	}
	input.Namespace = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Namespace = true
}

func (input *Input) SetResourcePath(value string) {
	if input == nil {
		return
	}
	input.ResourcePath = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ResourcePath = true
}
