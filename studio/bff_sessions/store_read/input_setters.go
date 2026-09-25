package store_read

func (input *Input) SetSessionIdHash(value string) {
	if input == nil {
		return
	}
	input.SessionIdHash = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.SessionIdHash = true
}
