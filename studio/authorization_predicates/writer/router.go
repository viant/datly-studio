package writer

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for authorization_predicate.
type AuthorizationPredicateComponent struct {
	Contract xdatly.Component[AuthorizationPredicateMutationInput, AuthorizationPredicateMutationOutput] "component:\"authorization_predicate,path=/v1/studio/security/authorization-predicates,method=PATCH,connector=studio,view=authorization_predicate\" routeName:\"authorization_predicate\" mutation:\"patch\" caseFormat:\"lc\""
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
