package writer

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for policy.
type PolicyComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"policy,path=/v1/studio/resource-policies,method=PATCH,connector=studio,view=policy\" routeName:\"policy\" mutation:\"patch\" caseFormat:\"lc\""
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
