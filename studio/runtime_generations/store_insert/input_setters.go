package store_insert

func (input *Input) SetGenerations(value []*StoredGeneration) {
	if input == nil {
		return
	}
	input.Generations = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Generations = true
}
