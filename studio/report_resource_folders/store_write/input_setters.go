package store_write

func (input *Input) SetFolders(value []*StoredFolder) {
	if input == nil {
		return
	}
	input.Folders = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Folders = true
}

func (input *Input) SetFolderKeys(value []FolderKeysRow) {
	if input == nil {
		return
	}
	input.FolderKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.FolderKeys = true
}

func (input *Input) SetCurrentFolder(value []*CurrentFolderView) {
	if input == nil {
		return
	}
	input.CurrentFolder = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentFolder = true
}
