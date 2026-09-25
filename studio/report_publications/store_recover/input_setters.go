package store_recover

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

func (input *Input) SetStagedGeneration(value int64) {
	if input == nil {
		return
	}
	input.StagedGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.StagedGeneration = true
}

func (input *Input) SetRestoreStatus(value string) {
	if input == nil {
		return
	}
	input.RestoreStatus = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.RestoreStatus = true
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
