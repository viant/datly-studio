package store_catalog

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

func (input *Input) SetState(value string) {
	if input == nil {
		return
	}
	input.State = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.State = true
}

func (input *Input) SetAuthoringMode(value string) {
	if input == nil {
		return
	}
	input.AuthoringMode = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.AuthoringMode = true
}

func (input *Input) SetCompileStatus(value string) {
	if input == nil {
		return
	}
	input.CompileStatus = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CompileStatus = true
}

func (input *Input) SetCreatedBy(value string) {
	if input == nil {
		return
	}
	input.CreatedBy = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CreatedBy = true
}

func (input *Input) SetLimit(value int) {
	if input == nil {
		return
	}
	input.Limit = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Limit = true
}

func (input *Input) SetOffset(value int) {
	if input == nil {
		return
	}
	input.Offset = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Offset = true
}
