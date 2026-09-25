package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for authorization_predicate.
type AuthorizationPredicateComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"authorization_predicate,path=/_studio/authorization-predicate-store,method=PATCH,connector=studio,view=authorization_predicate\" routeName:\"authorization_predicate\" mutation:\"patch\" caseFormat:\"lc\""
}

// AuthorizationPredicateDatlyType keeps the public component type linked for blank-import discovery.
func AuthorizationPredicateDatlyType() reflect.Type {
	return reflect.TypeOf((*AuthorizationPredicateComponent)(nil)).Elem()
}

// Datly anchors this package's public component contract.
var AuthorizationPredicateDatly = new(AuthorizationPredicateComponent)
var AuthorizationPredicateDatlyLinkedType = AuthorizationPredicateDatlyType()

func (AuthorizationPredicateComponent) EmbedFS() *embed.FS {
	return &AuthorizationPredicateDatlyResources
}

func (AuthorizationPredicateComponent) EmbedNamespace() string {
	return AuthorizationPredicateDatlyResourceNamespace
}
