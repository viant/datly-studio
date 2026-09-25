package store_stage

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for publication.
type PublicationComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"publication,path=/_studio/publication-store/stage,method=PATCH,connector=studio,view=publication\" routeName:\"publication\" mutation:\"patch\" caseFormat:\"lc\""
}

// PublicationDatlyType keeps the public component type linked for blank-import discovery.
func PublicationDatlyType() reflect.Type { return reflect.TypeOf((*PublicationComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var PublicationDatly = new(PublicationComponent)
var PublicationDatlyLinkedType = PublicationDatlyType()

func (PublicationComponent) EmbedFS() *embed.FS {
	return &PublicationDatlyResources
}

func (PublicationComponent) EmbedNamespace() string {
	return PublicationDatlyResourceNamespace
}
