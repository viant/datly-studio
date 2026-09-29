package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for file.
type FileComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"file,path=/_studio/resource-file-store/write,method=PATCH,connector=studio,view=file,internal=true\" routeName:\"file\" mutation:\"patch\" caseFormat:\"lc\""
}

// FileDatlyType returns the public component type.
func FileDatlyType() reflect.Type { return reflect.TypeOf((*FileComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var FileDatly = new(FileComponent)
var _datlyReachableFileComponent = reflect.TypeFor[FileComponent]()

func (FileComponent) EmbedFS() *embed.FS {
	return &FileDatlyResources
}

func (FileComponent) EmbedNamespace() string {
	return FileDatlyResourceNamespace
}
