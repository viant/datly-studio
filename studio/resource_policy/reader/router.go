package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for policy.
type PolicyComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"policy,path=/_studio/resource-policy-store/read,method=GET,connector=studio,view=policy,internal=true\" routeName:\"policy\" caseFormat:\"lc\""
}

// PolicyDatlyType keeps the public component type linked for blank-import discovery.
func PolicyDatlyType() reflect.Type { return reflect.TypeOf((*PolicyComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var PolicyDatly = new(PolicyComponent)
var PolicyDatlyLinkedType = PolicyDatlyType()

func (PolicyComponent) EmbedFS() *embed.FS {
	return &PolicyDatlyResources
}

func (PolicyComponent) EmbedNamespace() string {
	return PolicyDatlyResourceNamespace
}
