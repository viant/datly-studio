package store_state

func (input *Input) SetOperation(value string) {
	if input == nil {
		return
	}
	input.Operation = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Operation = true
}

func (input *Input) SetTargetGeneration(value int64) {
	if input == nil {
		return
	}
	input.TargetGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.TargetGeneration = true
}

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

func (input *Input) SetGenerationKeys(value []GenerationKeysRow) {
	if input == nil {
		return
	}
	input.GenerationKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.GenerationKeys = true
}

func (input *Input) SetCurrentGeneration(value []*CurrentGenerationView) {
	if input == nil {
		return
	}
	input.CurrentGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentGeneration = true
}
