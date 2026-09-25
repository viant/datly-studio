package store_catalog

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

func (input *Input) SetQuery(value string) {
	if input == nil {
		return
	}
	input.Query = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Query = true
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

func (input *Input) SetOwnerId(value string) {
	if input == nil {
		return
	}
	input.OwnerId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.OwnerId = true
}

func (input *Input) SetDriver(value string) {
	if input == nil {
		return
	}
	input.Driver = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Driver = true
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

func (input *Input) SetScoped(value bool) {
	if input == nil {
		return
	}
	input.Scoped = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Scoped = true
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
