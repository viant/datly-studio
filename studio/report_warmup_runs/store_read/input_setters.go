package store_read

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

func (input *Input) SetRunId(value string) {
	if input == nil {
		return
	}
	input.RunId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.RunId = true
}

func (input *Input) SetActiveKey(value string) {
	if input == nil {
		return
	}
	input.ActiveKey = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ActiveKey = true
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

func (input *Input) SetPageLimit(value int) {
	if input == nil {
		return
	}
	input.PageLimit = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.PageLimit = true
}

func (input *Input) SetPageOffset(value int) {
	if input == nil {
		return
	}
	input.PageOffset = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.PageOffset = true
}
