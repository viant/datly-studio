package store_capabilities

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for capability.
type CapabilityComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"capability,path=/_studio/report-capabilities,method=GET,connector=studio,view=capability\" routeName:\"capability\" caseFormat:\"lc\""
}

// CapabilityDatlyType keeps the public component type linked for blank-import discovery.
func CapabilityDatlyType() reflect.Type { return reflect.TypeOf((*CapabilityComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var CapabilityDatly = new(CapabilityComponent)
var CapabilityDatlyLinkedType = CapabilityDatlyType()

func (CapabilityComponent) EmbedFS() *embed.FS {
	return &CapabilityDatlyResources
}

func (CapabilityComponent) EmbedNamespace() string {
	return CapabilityDatlyResourceNamespace
}
