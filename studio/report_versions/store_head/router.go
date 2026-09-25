package store_head

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for head.
type HeadComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"head,path=/_studio/report-version-store/head,method=GET,connector=studio,view=head\" routeName:\"head\" caseFormat:\"lc\""
}

// HeadDatlyType keeps the public component type linked for blank-import discovery.
func HeadDatlyType() reflect.Type { return reflect.TypeOf((*HeadComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var HeadDatly = new(HeadComponent)
var HeadDatlyLinkedType = HeadDatlyType()

func (HeadComponent) EmbedFS() *embed.FS {
	return &HeadDatlyResources
}

func (HeadComponent) EmbedNamespace() string {
	return HeadDatlyResourceNamespace
}
