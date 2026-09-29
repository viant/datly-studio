package presence

func (input *Input) SetTenant(value string) {
	if input == nil {
		return
	}
	input.Tenant = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Tenant = true
}

func (input *Input) SetResourceKind(value string) {
	if input == nil {
		return
	}
	input.ResourceKind = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ResourceKind = true
}

func (input *Input) SetResourceId(value string) {
	if input == nil {
		return
	}
	input.ResourceId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ResourceId = true
}
