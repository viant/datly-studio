package store_active

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for definition.
type DefinitionComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"definition,path=/_studio/runtime-definitions/active,method=GET,connector=studio,view=definition\" routeName:\"definition\" caseFormat:\"lc\""
}

// DefinitionDatlyType keeps the public component type linked for blank-import discovery.
func DefinitionDatlyType() reflect.Type { return reflect.TypeOf((*DefinitionComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var DefinitionDatly = new(DefinitionComponent)
var DefinitionDatlyLinkedType = DefinitionDatlyType()

func (DefinitionComponent) EmbedFS() *embed.FS {
	return &DefinitionDatlyResources
}

func (DefinitionComponent) EmbedNamespace() string {
	return DefinitionDatlyResourceNamespace
}
