package store_read

// Output is the generated output scaffold for authorization_predicate.
type Output struct {
	AuthorizationPredicates []*StoredAuthorizationPredicate `parameter:"AuthorizationPredicates,kind=output,in=view,dataType=[]*StoredAuthorizationPredicate" view:"authorization_predicate,type=StoredAuthorizationPredicate,table=authorization_predicates,limit=500,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={updated_at,name}" sql:"uri=studio_authorization_predicates_store_read_authorization_predicate:sql/read.sql"`
}
