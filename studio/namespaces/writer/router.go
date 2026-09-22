package writer

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for namespace.
type NamespaceComponent struct {
	Contract xdatly.Component[NamespaceMutationInput, NamespaceMutationOutput] "component:\"namespace,path=/v1/studio/namespaces,method=PATCH,connector=studio,view=namespace\" routeName:\"namespace\" mutation:\"patch\" caseFormat:\"lc\""
}

// NamespaceDatlyType keeps the public component type linked for blank-import discovery.
func NamespaceDatlyType() reflect.Type { return reflect.TypeOf((*NamespaceComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var NamespaceDatly = new(NamespaceComponent)
var NamespaceDatlyLinkedType = NamespaceDatlyType()

func (NamespaceComponent) EmbedFS() *embed.FS {
	return &NamespaceDatlyResources
}

func (NamespaceComponent) EmbedNamespace() string {
	return NamespaceDatlyResourceNamespace
}
