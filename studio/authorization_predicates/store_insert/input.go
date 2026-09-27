package store_insert

// Input is the generated input scaffold for authorization_predicate.
type Input struct {
	AuthorizationPredicates []*StoredAuthorizationPredicate `parameter:"AuthorizationPredicates,kind=body,in=data,dataType=[]*StoredAuthorizationPredicate" view:"authorization_predicate,type=StoredAuthorizationPredicate,entityHooks=PredicateInsertRules,table=authorization_predicates" sql:"uri=studio_authorization_predicates_store_insert_authorization_predicate:sql/read.sql"`
	Has                     *InputHas                       `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	AuthorizationPredicates bool
}
