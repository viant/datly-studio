package store_write

func (input *Input) SetFiles(value []*StoredFile) {
	if input == nil {
		return
	}
	input.Files = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Files = true
}

func (input *Input) SetFileKeys(value []FileKeysRow) {
	if input == nil {
		return
	}
	input.FileKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.FileKeys = true
}

func (input *Input) SetCurrentFile(value []*CurrentFileView) {
	if input == nil {
		return
	}
	input.CurrentFile = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentFile = true
}
