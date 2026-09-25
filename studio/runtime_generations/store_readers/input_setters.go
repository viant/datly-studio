package store_readers

func (input *Input) SetGeneration(value int64) {
	if input == nil {
		return
	}
	input.Generation = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Generation = true
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
