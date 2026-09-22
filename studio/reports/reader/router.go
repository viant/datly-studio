package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for report.
type ReportComponent struct {
	Contract1 xdatly.Component[Input, Output] "component:\"report,path=/v1/studio/reports,method=GET,connector=studio,view=report\" routeName:\"report\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.reports.read\\\",\\\"description\\\":\\\"Read Datly Studio reports\\\"}]\" caseFormat:\"lc\""
	Contract2 xdatly.Component[Input, Output] "component:\"report,path=/v1/studio/reports/{id},method=GET,connector=studio,view=report\" routeName:\"report\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.reports.readById\\\",\\\"description\\\":\\\"Read Datly Studio reports\\\"}]\" caseFormat:\"lc\""
}

// ReportDatlyType keeps the public component type linked for blank-import discovery.
func ReportDatlyType() reflect.Type { return reflect.TypeOf((*ReportComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ReportDatly = new(ReportComponent)
var ReportDatlyLinkedType = ReportDatlyType()

func (ReportComponent) EmbedFS() *embed.FS {
	return &ReportDatlyResources
}

func (ReportComponent) EmbedNamespace() string {
	return ReportDatlyResourceNamespace
}
