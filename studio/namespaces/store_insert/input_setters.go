package store_insert

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
