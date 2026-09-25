package store_list

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

func (input *Input) SetOperation(value string) {
	if input == nil {
		return
	}
	input.Operation = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Operation = true
}

func (input *Input) SetStatus(value string) {
	if input == nil {
		return
	}
	input.Status = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Status = true
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
