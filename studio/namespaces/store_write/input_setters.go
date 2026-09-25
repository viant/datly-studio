package store_write

func (input *Input) SetNamespaces(value []*StoredNamespace) {
	if input == nil {
		return
	}
	input.Namespaces = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Namespaces = true
}

func (input *Input) SetNamespaceKeys(value []NamespaceKeysRow) {
	if input == nil {
		return
	}
	input.NamespaceKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.NamespaceKeys = true
}

func (input *Input) SetCurrentNamespace(value []*CurrentNamespaceView) {
	if input == nil {
		return
	}
	input.CurrentNamespace = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentNamespace = true
}
