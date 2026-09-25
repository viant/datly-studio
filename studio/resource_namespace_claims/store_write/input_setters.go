package store_write

func (input *Input) SetOperation(value string) {
	if input == nil {
		return
	}
	input.Operation = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Operation = true
}

func (input *Input) SetClaims(value []*StoredClaim) {
	if input == nil {
		return
	}
	input.Claims = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Claims = true
}

func (input *Input) SetClaimKeys(value []ClaimKeysRow) {
	if input == nil {
		return
	}
	input.ClaimKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ClaimKeys = true
}

func (input *Input) SetCurrentClaim(value []*CurrentClaimView) {
	if input == nil {
		return
	}
	input.CurrentClaim = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentClaim = true
}
