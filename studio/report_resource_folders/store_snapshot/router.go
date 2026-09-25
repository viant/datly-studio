package store_snapshot

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for folder.
type FolderComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"folder,path=/_studio/resource-snapshot/folders,method=GET,connector=studio,view=folder\" routeName:\"folder\" caseFormat:\"lc\""
}

// FolderDatlyType keeps the public component type linked for blank-import discovery.
func FolderDatlyType() reflect.Type { return reflect.TypeOf((*FolderComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var FolderDatly = new(FolderComponent)
var FolderDatlyLinkedType = FolderDatlyType()

func (FolderComponent) EmbedFS() *embed.FS {
	return &FolderDatlyResources
}

func (FolderComponent) EmbedNamespace() string {
	return FolderDatlyResourceNamespace
}
