package writer

func (input *Input) SetPolicies(value []*ResourcePolicyHead) {
	if input == nil {
		return
	}
	input.Policies = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Policies = true
}

func (input *Input) SetPolicyKeys(value []PolicyKeysRow) {
	if input == nil {
		return
	}
	input.PolicyKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.PolicyKeys = true
}

func (input *Input) SetCurrentPolicy(value []*CurrentPolicyView) {
	if input == nil {
		return
	}
	input.CurrentPolicy = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentPolicy = true
}

func (input *Input) SetCurrentHistory(value []*CurrentHistoryView) {
	if input == nil {
		return
	}
	input.CurrentHistory = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentHistory = true
}
