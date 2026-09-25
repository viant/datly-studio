package store_insert

func (input *Input) SetVersions(value []*StoredVersion) {
	if input == nil {
		return
	}
	input.Versions = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Versions = true
}
