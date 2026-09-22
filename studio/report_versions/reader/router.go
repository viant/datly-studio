package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for version.
type VersionComponent struct {
	Contract1 xdatly.Component[Input, Output] "component:\"version,path=/v1/studio/reports/{reportId}/versions,method=GET,connector=studio,view=version\" routeName:\"version\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_versions.read\\\",\\\"description\\\":\\\"Read immutable Datly Studio report versions\\\"}]\" caseFormat:\"lc\""
	Contract2 xdatly.Component[Input, Output] "component:\"version,path=/v1/studio/reports/{reportId}/versions/{versionNo},method=GET,connector=studio,view=version\" routeName:\"version\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_versions.readByVersionNo\\\",\\\"description\\\":\\\"Read immutable Datly Studio report versions\\\"}]\" caseFormat:\"lc\""
}

// VersionDatlyType keeps the public component type linked for blank-import discovery.
func VersionDatlyType() reflect.Type { return reflect.TypeOf((*VersionComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var VersionDatly = new(VersionComponent)
var VersionDatlyLinkedType = VersionDatlyType()

func (VersionComponent) EmbedFS() *embed.FS {
	return &VersionDatlyResources
}

func (VersionComponent) EmbedNamespace() string {
	return VersionDatlyResourceNamespace
}
