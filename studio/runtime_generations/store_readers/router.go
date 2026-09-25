package store_readers

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for reader.
type ReaderComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"reader,path=/_studio/runtime-generation-store/readers,method=GET,connector=studio,view=reader\" routeName:\"reader\" caseFormat:\"lc\""
}

// ReaderDatlyType keeps the public component type linked for blank-import discovery.
func ReaderDatlyType() reflect.Type { return reflect.TypeOf((*ReaderComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ReaderDatly = new(ReaderComponent)
var ReaderDatlyLinkedType = ReaderDatlyType()

func (ReaderComponent) EmbedFS() *embed.FS {
	return &ReaderDatlyResources
}

func (ReaderComponent) EmbedNamespace() string {
	return ReaderDatlyResourceNamespace
}
