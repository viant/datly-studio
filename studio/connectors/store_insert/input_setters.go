package store_insert

func (input *Input) SetConnectors(value []*StoredConnector) {
	if input == nil {
		return
	}
	input.Connectors = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Connectors = true
}
