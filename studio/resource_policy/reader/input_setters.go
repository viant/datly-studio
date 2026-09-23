package reader

func (input *Input) SetTenantId(value string) {
	if input == nil {
		return
	}
	input.TenantId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.TenantId = true
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

func (input *Input) SetResourceVersion(value string) {
	if input == nil {
		return
	}
	input.ResourceVersion = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ResourceVersion = true
}
