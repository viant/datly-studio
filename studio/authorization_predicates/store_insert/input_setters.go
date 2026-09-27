package store_insert

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
