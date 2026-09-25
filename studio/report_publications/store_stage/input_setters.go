package store_stage

func (input *Input) SetNextGeneration(value int64) {
	if input == nil {
		return
	}
	input.NextGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.NextGeneration = true
}

func (input *Input) SetPublications(value []*StoredPublication) {
	if input == nil {
		return
	}
	input.Publications = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Publications = true
}

func (input *Input) SetPublicationKeys(value []PublicationKeysRow) {
	if input == nil {
		return
	}
	input.PublicationKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.PublicationKeys = true
}

func (input *Input) SetCurrentPublication(value []*CurrentPublicationView) {
	if input == nil {
		return
	}
	input.CurrentPublication = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentPublication = true
}
