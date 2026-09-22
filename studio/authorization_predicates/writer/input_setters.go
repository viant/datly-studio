package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *AuthorizationPredicateMutationInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateMutationInputHas{}
	}
	input.Has.Jwt = true
}

func (input *AuthorizationPredicateMutationInput) SetAuthorizationPredicates(value []*AuthorizationPredicateMutationRecord) {
	if input == nil {
		return
	}
	input.AuthorizationPredicates = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateMutationInputHas{}
	}
	input.Has.AuthorizationPredicates = true
}

func (input *AuthorizationPredicateMutationInput) SetAuthorizationPredicateKeys(value []AuthorizationPredicateKeysRow) {
	if input == nil {
		return
	}
	input.AuthorizationPredicateKeys = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateMutationInputHas{}
	}
	input.Has.AuthorizationPredicateKeys = true
}

func (input *AuthorizationPredicateMutationInput) SetCurrentAuthorizationPredicate(value []*CurrentAuthorizationPredicateView) {
	if input == nil {
		return
	}
	input.CurrentAuthorizationPredicate = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateMutationInputHas{}
	}
	input.Has.CurrentAuthorizationPredicate = true
}
