package store_catalog

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for generation.
type GenerationComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"generation,path=/_studio/runtime-generation-store/catalog,method=GET,connector=studio,view=generation\" routeName:\"generation\" caseFormat:\"lc\""
}

// GenerationDatlyType keeps the public component type linked for blank-import discovery.
func GenerationDatlyType() reflect.Type { return reflect.TypeOf((*GenerationComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var GenerationDatly = new(GenerationComponent)
var GenerationDatlyLinkedType = GenerationDatlyType()

func (GenerationComponent) EmbedFS() *embed.FS {
	return &GenerationDatlyResources
}

func (GenerationComponent) EmbedNamespace() string {
	return GenerationDatlyResourceNamespace
}
