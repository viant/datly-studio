package store_usage

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
