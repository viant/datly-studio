package store_write

func (input *Input) SetRuns(value []*StoredWarmupRun) {
	if input == nil {
		return
	}
	input.Runs = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Runs = true
}

func (input *Input) SetWarmupRunKeys(value []WarmupRunKeysRow) {
	if input == nil {
		return
	}
	input.WarmupRunKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.WarmupRunKeys = true
}

func (input *Input) SetCurrentWarmupRun(value []*CurrentWarmupRunView) {
	if input == nil {
		return
	}
	input.CurrentWarmupRun = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentWarmupRun = true
}
