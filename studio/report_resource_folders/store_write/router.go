package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for folder.
type FolderComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"folder,path=/_studio/resource-folder-store/write,method=PATCH,connector=studio,view=folder,internal=true\" routeName:\"folder\" mutation:\"patch\" caseFormat:\"lc\""
}

// FolderDatlyType returns the public component type.
func FolderDatlyType() reflect.Type { return reflect.TypeOf((*FolderComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var FolderDatly = new(FolderComponent)
var _datlyReachableFolderComponent = reflect.TypeFor[FolderComponent]()

func (FolderComponent) EmbedFS() *embed.FS {
	return &FolderDatlyResources
}

func (FolderComponent) EmbedNamespace() string {
	return FolderDatlyResourceNamespace
}
