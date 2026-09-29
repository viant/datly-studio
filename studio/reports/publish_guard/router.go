package publish_guard

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for report.
type ReportComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"report,path=/_studio/reports/publish-guard,method=POST,connector=studio,view=report,internal=true\" routeName:\"report\" caseFormat:\"lc\""
}

// ReportDatlyType returns the public component type.
func ReportDatlyType() reflect.Type { return reflect.TypeOf((*ReportComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var ReportDatly = new(ReportComponent)
var _datlyReachableReportComponent = reflect.TypeFor[ReportComponent]()

func (ReportComponent) EmbedFS() *embed.FS {
	return &ReportDatlyResources
}

func (ReportComponent) EmbedNamespace() string {
	return ReportDatlyResourceNamespace
}
