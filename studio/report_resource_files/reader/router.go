package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for file.
type FileComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"file,path=/v1/studio/reports/{reportId}/versions/{versionNo}/resources/files,method=GET,connector=studio,view=file\" routeName:\"file\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_resource_files.read\\\",\\\"description\\\":\\\"Read report resource files\\\"}]\" caseFormat:\"lc\""
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
