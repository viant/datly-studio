package store_catalog

func (input *Input) SetGenerationNo(value int64) {
	if input == nil {
		return
	}
	input.GenerationNo = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.GenerationNo = true
}

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
