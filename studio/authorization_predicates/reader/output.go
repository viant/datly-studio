package reader

// AuthorizationPredicateQueryOutput is the generated output scaffold for authorization_predicate.
type AuthorizationPredicateQueryOutput struct {
	AuthorizationPredicates []*AuthorizationPredicateRecord `parameter:"AuthorizationPredicates,kind=output,in=view,dataType=[]*AuthorizationPredicateRecord" view:"authorization_predicate,type=AuthorizationPredicateRecord,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={name,title,package_path,type_name,status,updated_at}" sql:"uri=studio_authorization_predicates_reader_authorization_predicate:sql/read.sql"`
}
