package store_staged

func (input *Input) SetDesiredGeneration(value int64) {
	if input == nil {
		return
	}
	input.DesiredGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.DesiredGeneration = true
}
