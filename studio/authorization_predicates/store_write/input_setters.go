package store_write

func (input *Input) SetAuthorizationPredicates(value []*StoredAuthorizationPredicate) {
	if input == nil {
		return
	}
	input.AuthorizationPredicates = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.AuthorizationPredicates = true
}

func (input *Input) SetAuthorizationPredicateKeys(value []AuthorizationPredicateKeysRow) {
	if input == nil {
		return
	}
	input.AuthorizationPredicateKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.AuthorizationPredicateKeys = true
}

func (input *Input) SetCurrentAuthorizationPredicate(value []*CurrentAuthorizationPredicateView) {
	if input == nil {
		return
	}
	input.CurrentAuthorizationPredicate = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentAuthorizationPredicate = true
}
