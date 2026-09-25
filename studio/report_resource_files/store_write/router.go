package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for file.
type FileComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"file,path=/_studio/resource-file-store/write,method=PATCH,connector=studio,view=file\" routeName:\"file\" mutation:\"patch\" caseFormat:\"lc\""
}

// FileDatlyType keeps the public component type linked for blank-import discovery.
func FileDatlyType() reflect.Type { return reflect.TypeOf((*FileComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var FileDatly = new(FileComponent)
var FileDatlyLinkedType = FileDatlyType()

func (FileComponent) EmbedFS() *embed.FS {
	return &FileDatlyResources
}

func (FileComponent) EmbedNamespace() string {
	return FileDatlyResourceNamespace
}
