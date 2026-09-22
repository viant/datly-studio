package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for warmup_run.
type WarmupRunComponent struct {
	Contract xdatly.Component[WarmupRunInput, WarmupRunOutput] "component:\"warmup_run,path=/v1/studio/report-warmup-runs,method=GET,connector=studio,view=warmup_run\" routeName:\"warmup_run\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_warmup_runs.read\\\",\\\"description\\\":\\\"Read durable cache warmup run evidence\\\"}]\" caseFormat:\"lc\""
}

// WarmupRunDatlyType keeps the public component type linked for blank-import discovery.
func WarmupRunDatlyType() reflect.Type { return reflect.TypeOf((*WarmupRunComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var WarmupRunDatly = new(WarmupRunComponent)
var WarmupRunDatlyLinkedType = WarmupRunDatlyType()

func (WarmupRunComponent) EmbedFS() *embed.FS {
	return &WarmupRunDatlyResources
}

func (WarmupRunComponent) EmbedNamespace() string {
	return WarmupRunDatlyResourceNamespace
}
