package store_expired

import (
	time "time"
)

func (input *Input) SetBefore(value time.Time) {
	if input == nil {
		return
	}
	input.Before = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Before = true
}

func (input *Input) SetPageLimit(value int) {
	if input == nil {
		return
	}
	input.PageLimit = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.PageLimit = true
}
