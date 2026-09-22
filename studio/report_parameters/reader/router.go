package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for parameter.
type ParameterComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"parameter,path=/v1/studio/reports/{reportId}/versions/{versionNo}/parameters,method=GET,connector=studio,view=parameter\" routeName:\"parameter\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_parameters.read\\\",\\\"description\\\":\\\"Read report parameters and predicates\\\"}]\" caseFormat:\"lc\""
}

// ParameterDatlyType keeps the public component type linked for blank-import discovery.
func ParameterDatlyType() reflect.Type { return reflect.TypeOf((*ParameterComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ParameterDatly = new(ParameterComponent)
var ParameterDatlyLinkedType = ParameterDatlyType()

func (ParameterComponent) EmbedFS() *embed.FS {
	return &ParameterDatlyResources
}

func (ParameterComponent) EmbedNamespace() string {
	return ParameterDatlyResourceNamespace
}
