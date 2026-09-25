package store_write

func (input *Input) SetAccess(value []*StoredACL) {
	if input == nil {
		return
	}
	input.Access = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Access = true
}

func (input *Input) SetAclKeys(value []AclKeysRow) {
	if input == nil {
		return
	}
	input.AclKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.AclKeys = true
}

func (input *Input) SetCurrentAcl(value []*CurrentAclView) {
	if input == nil {
		return
	}
	input.CurrentAcl = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentAcl = true
}
