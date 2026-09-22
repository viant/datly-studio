package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// AuthorizationPredicateMutationInput is the generated input scaffold for authorization_predicate.
type AuthorizationPredicateMutationInput struct {
	Jwt                                       *jwt.Claims                               `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	AuthorizationPredicates                   []*AuthorizationPredicateMutationRecord   `parameter:"AuthorizationPredicates,kind=body,in=data,dataType=[]*AuthorizationPredicateMutationRecord" view:"authorization_predicate,type=AuthorizationPredicateMutationRecord,table=authorization_predicates" sql:"uri=studio_authorization_predicates_writer_authorization_predicate:sql/patch.sql"`
	AuthorizationPredicateKeys                []AuthorizationPredicateKeysRow           `parameter:"AuthorizationPredicateKeys,kind=param,in=AuthorizationPredicates,cardinality=Many" codec:"structql,'uri=studio_authorization_predicates_writer_authorization_predicate:sql/authorization_predicate_keys.sql'"`
	CurrentAuthorizationPredicate             []*CurrentAuthorizationPredicateView      `parameter:"CurrentAuthorizationPredicate,kind=view,in=CurrentAuthorizationPredicate,cardinality=Many" view:"CurrentAuthorizationPredicate,table=authorization_predicates" sql:"uri=studio_authorization_predicates_writer_authorization_predicate:sql/current_authorization_predicate.sql"`
	_authorizationPredicateHandlerReadIndexes *AuthorizationPredicateHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                                       *AuthorizationPredicateMutationInputHas   `setMarker:"true" typeName:"AuthorizationPredicateMutationInputHas" json:"-" sqlx:"-"`
}

type AuthorizationPredicateMutationInputHas struct {
	Jwt                           bool
	AuthorizationPredicates       bool
	AuthorizationPredicateKeys    bool
	CurrentAuthorizationPredicate bool
}
