package store_write

// Input is the generated input scaffold for authorization_predicate.
type Input struct {
	AuthorizationPredicates                   []*StoredAuthorizationPredicate           `parameter:"AuthorizationPredicates,kind=body,in=data,dataType=[]*StoredAuthorizationPredicate" view:"authorization_predicate,type=StoredAuthorizationPredicate,entityHooks=PredicateStoreRules,table=authorization_predicates" sql:"uri=studio_authorization_predicates_store_write_authorization_predicate:sql/patch.sql"`
	AuthorizationPredicateKeys                []AuthorizationPredicateKeysRow           `parameter:"AuthorizationPredicateKeys,kind=param,in=AuthorizationPredicates,cardinality=Many" codec:"structql,'uri=studio_authorization_predicates_store_write_authorization_predicate:sql/authorization_predicate_keys.sql'"`
	CurrentAuthorizationPredicate             []*CurrentAuthorizationPredicateView      `parameter:"CurrentAuthorizationPredicate,kind=view,in=CurrentAuthorizationPredicate,cardinality=Many" view:"CurrentAuthorizationPredicate,table=authorization_predicates" sql:"uri=studio_authorization_predicates_store_write_authorization_predicate:sql/current_authorization_predicate.sql"`
	_authorizationPredicateHandlerReadIndexes *AuthorizationPredicateHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                                       *InputHas                                 `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	AuthorizationPredicates       bool
	AuthorizationPredicateKeys    bool
	CurrentAuthorizationPredicate bool
}
