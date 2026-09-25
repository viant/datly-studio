package store_validation

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

func (input *Input) SetVersionKeys(value []VersionKeysRow) {
	if input == nil {
		return
	}
	input.VersionKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.VersionKeys = true
}

func (input *Input) SetCurrentVersion(value []*CurrentVersionView) {
	if input == nil {
		return
	}
	input.CurrentVersion = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentVersion = true
}
