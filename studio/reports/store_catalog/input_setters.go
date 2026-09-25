package store_catalog

func (input *Input) SetId(value string) {
	if input == nil {
		return
	}
	input.Id = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Id = true
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

func (input *Input) SetConnectorName(value string) {
	if input == nil {
		return
	}
	input.ConnectorName = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ConnectorName = true
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
