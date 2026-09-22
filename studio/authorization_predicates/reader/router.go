package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for authorization_predicate.
type AuthorizationPredicateComponent struct {
	Contract1 xdatly.Component[AuthorizationPredicateQueryInput, AuthorizationPredicateQueryOutput] "component:\"authorization_predicate,path=/v1/studio/security/authorization-predicates,method=GET,connector=studio,view=authorization_predicate\" routeName:\"authorization_predicate\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.security.authorization_predicates.read\\\",\\\"description\\\":\\\"Read governed authorization predicate links\\\"}]\" caseFormat:\"lc\""
	Contract2 xdatly.Component[AuthorizationPredicateQueryInput, AuthorizationPredicateQueryOutput] "component:\"authorization_predicate,path=/v1/studio/security/authorization-predicates/{name},method=GET,connector=studio,view=authorization_predicate\" routeName:\"authorization_predicate\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.security.authorization_predicates.readByName\\\",\\\"description\\\":\\\"Read governed authorization predicate links\\\"}]\" caseFormat:\"lc\""
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
