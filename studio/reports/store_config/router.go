package store_config

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for report.
type ReportComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"report,path=/_studio/report-store/config,method=PATCH,connector=studio,view=report\" routeName:\"report\" mutation:\"patch\" caseFormat:\"lc\""
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
