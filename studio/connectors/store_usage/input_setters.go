package store_usage

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
