package export_dql

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for version.
type VersionComponent struct {
	Contract xdatly.Component[VersionExportInput, VersionExportOutput] "component:\"version,path=/v1/studio/sdk/versions.export_dql,method=POST,connector=studio,view=version\" routeName:\"version\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.sdk.versions.export_dql\\\",\\\"description\\\":\\\"Export authorized Datly Studio DQL source\\\"}]\" caseFormat:\"lc\""
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
