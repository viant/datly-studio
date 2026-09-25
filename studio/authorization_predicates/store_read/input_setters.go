package store_read

func (input *Input) SetName(value string) {
	if input == nil {
		return
	}
	input.Name = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Name = true
}

func (input *Input) SetSearchPattern(value string) {
	if input == nil {
		return
	}
	input.SearchPattern = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SearchPattern = true
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

func (input *Input) SetOrderBy(value string) {
	if input == nil {
		return
	}
	input.OrderBy = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.OrderBy = true
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
