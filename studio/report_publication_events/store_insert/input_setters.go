package store_insert

func (input *Input) SetEvents(value []*StoredEvent) {
	if input == nil {
		return
	}
	input.Events = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Events = true
}
