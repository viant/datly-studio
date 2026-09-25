package store_runtime_catalog

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

func (input *Input) SetLive(value bool) {
	if input == nil {
		return
	}
	input.Live = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Live = true
}
