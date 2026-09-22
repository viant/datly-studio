package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for exposure.
type ExposureComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"exposure,path=/v1/studio/reports/{reportId}/versions/{versionNo}/mcp,method=GET,connector=studio,view=exposure\" routeName:\"exposure\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_mcp_exposures.read\\\",\\\"description\\\":\\\"Read report MCP exposures\\\"}]\" caseFormat:\"lc\""
}

// ExposureDatlyType keeps the public component type linked for blank-import discovery.
func ExposureDatlyType() reflect.Type { return reflect.TypeOf((*ExposureComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ExposureDatly = new(ExposureComponent)
var ExposureDatlyLinkedType = ExposureDatlyType()

func (ExposureComponent) EmbedFS() *embed.FS {
	return &ExposureDatlyResources
}

func (ExposureComponent) EmbedNamespace() string {
	return ExposureDatlyResourceNamespace
}
