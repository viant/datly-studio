package store_active_others

func (input *Input) SetExcludeGeneration(value int64) {
	if input == nil {
		return
	}
	input.ExcludeGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ExcludeGeneration = true
}
