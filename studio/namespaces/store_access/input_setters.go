package store_access

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

func (input *Input) SetPermission(value string) {
	if input == nil {
		return
	}
	input.Permission = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Permission = true
}
