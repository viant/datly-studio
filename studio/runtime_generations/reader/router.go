package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for generation.
type GenerationComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"generation,path=/v1/studio/runtime-generations,method=GET,connector=studio,view=generation\" routeName:\"generation\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.runtime_generations.read\\\",\\\"description\\\":\\\"Read runtime generations\\\"}]\" caseFormat:\"lc\""
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
