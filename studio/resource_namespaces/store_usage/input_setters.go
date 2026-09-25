package store_usage

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
