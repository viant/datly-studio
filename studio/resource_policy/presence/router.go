package presence

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for head.
type HeadComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"head,path=/_studio/resource-policy-presence,method=GET,connector=studio,view=head,internal=true\" routeName:\"head\""
}

// HeadDatlyType returns the public component type.
func HeadDatlyType() reflect.Type { return reflect.TypeOf((*HeadComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var HeadDatly = new(HeadComponent)
var _datlyReachableHeadComponent = reflect.TypeFor[HeadComponent]()

func (HeadComponent) EmbedFS() *embed.FS {
	return &HeadDatlyResources
}

func (HeadComponent) EmbedNamespace() string {
	return HeadDatlyResourceNamespace
}
