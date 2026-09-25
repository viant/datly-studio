package store_status

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
