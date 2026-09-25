package store_expired

func (input *Input) SetExpiresBefore(value int64) {
	if input == nil {
		return
	}
	input.ExpiresBefore = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ExpiresBefore = true
}
