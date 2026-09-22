package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for namespace.
type NamespaceComponent struct {
	Contract1 xdatly.Component[NamespaceQueryInput, NamespaceQueryOutput] "component:\"namespace,path=/v1/studio/namespaces,method=GET,connector=studio,view=namespace\" routeName:\"namespace\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.namespaces.read\\\",\\\"description\\\":\\\"Read governed Datly Studio namespaces\\\"}]\" caseFormat:\"lc\""
	Contract2 xdatly.Component[NamespaceQueryInput, NamespaceQueryOutput] "component:\"namespace,path=/v1/studio/namespaces/{name},method=GET,connector=studio,view=namespace\" routeName:\"namespace\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.namespaces.readByName\\\",\\\"description\\\":\\\"Read governed Datly Studio namespaces\\\"}]\" caseFormat:\"lc\""
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
