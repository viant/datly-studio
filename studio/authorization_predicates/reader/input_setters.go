package reader

import (
	jwt "github.com/viant/scy/auth/jwt"
)

func (input *AuthorizationPredicateQueryInput) SetJwt(value *jwt.Claims) {
	if input == nil {
		return
	}
	input.Jwt = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Jwt = true
}

func (input *AuthorizationPredicateQueryInput) SetName(value string) {
	if input == nil {
		return
	}
	input.Name = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Name = true
}

func (input *AuthorizationPredicateQueryInput) SetQuery(value string) {
	if input == nil {
		return
	}
	input.Query = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Query = true
}

func (input *AuthorizationPredicateQueryInput) SetStatus(value string) {
	if input == nil {
		return
	}
	input.Status = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Status = true
}

func (input *AuthorizationPredicateQueryInput) SetFields(value []string) {
	if input == nil {
		return
	}
	input.Fields = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Fields = true
}

func (input *AuthorizationPredicateQueryInput) SetOrderBy(value string) {
	if input == nil {
		return
	}
	input.OrderBy = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.OrderBy = true
}

func (input *AuthorizationPredicateQueryInput) SetLimit(value int) {
	if input == nil {
		return
	}
	input.Limit = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Limit = true
}

func (input *AuthorizationPredicateQueryInput) SetOffset(value int) {
	if input == nil {
		return
	}
	input.Offset = value
	if input.Has == nil {
		input.Has = &AuthorizationPredicateQueryInputHas{}
	}
	input.Has.Offset = true
}
