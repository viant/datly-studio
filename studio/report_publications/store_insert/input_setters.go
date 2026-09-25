package store_insert

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
