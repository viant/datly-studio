package store_insert

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for namespace.
type NamespaceComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"namespace,path=/_studio/namespace-store/insert,method=POST,connector=studio,view=namespace\" routeName:\"namespace\" mutation:\"post\" caseFormat:\"lc\""
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
