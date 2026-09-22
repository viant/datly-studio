package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for view.
type ViewComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"view,path=/v1/studio/reports/{reportId}/versions/{versionNo}/views,method=GET,connector=studio,view=view\" routeName:\"view\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_views.read\\\",\\\"description\\\":\\\"Read compiled report views and fields\\\"}]\" caseFormat:\"lc\""
}

// ViewDatlyType keeps the public component type linked for blank-import discovery.
func ViewDatlyType() reflect.Type { return reflect.TypeOf((*ViewComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ViewDatly = new(ViewComponent)
var ViewDatlyLinkedType = ViewDatlyType()

func (ViewComponent) EmbedFS() *embed.FS {
	return &ViewDatlyResources
}

func (ViewComponent) EmbedNamespace() string {
	return ViewDatlyResourceNamespace
}
