package writer

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for exposure.
type ExposureComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"exposure,path=/v1/studio/report-mcp-exposures,method=PATCH,connector=studio,view=exposure\" routeName:\"exposure\" mutation:\"patch\" caseFormat:\"lc\""
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
