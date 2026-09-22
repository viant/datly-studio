package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for context.
type ContextComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"context,path=/v1/studio/auth/context,method=GET,connector=studio,view=context\" routeName:\"context\" caseFormat:\"lc\""
}

// ContextDatlyType keeps the public component type linked for blank-import discovery.
func ContextDatlyType() reflect.Type { return reflect.TypeOf((*ContextComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ContextDatly = new(ContextComponent)
var ContextDatlyLinkedType = ContextDatlyType()

func (ContextComponent) EmbedFS() *embed.FS {
	return &ContextDatlyResources
}

func (ContextComponent) EmbedNamespace() string {
	return ContextDatlyResourceNamespace
}
