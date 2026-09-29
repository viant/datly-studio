package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for warmup_run.
type WarmupRunComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"warmup_run,path=/_studio/report-warmup-run-store/write,method=PATCH,connector=studio,view=warmup_run,internal=true\" routeName:\"warmup_run\" mutation:\"patch\" caseFormat:\"lc\""
}

// WarmupRunDatlyType returns the public component type.
func WarmupRunDatlyType() reflect.Type { return reflect.TypeOf((*WarmupRunComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var WarmupRunDatly = new(WarmupRunComponent)
var _datlyReachableWarmupRunComponent = reflect.TypeFor[WarmupRunComponent]()

func (WarmupRunComponent) EmbedFS() *embed.FS {
	return &WarmupRunDatlyResources
}

func (WarmupRunComponent) EmbedNamespace() string {
	return WarmupRunDatlyResourceNamespace
}
