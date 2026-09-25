package store_import

func (input *Input) SetVersions(value []*ImportedVersion) {
	if input == nil {
		return
	}
	input.Versions = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Versions = true
}
