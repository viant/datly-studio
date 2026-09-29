package get

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for warmup_run.
type WarmupRunComponent struct {
	Contract xdatly.Component[WarmupGetInput, WarmupGetOutput] "component:\"warmup_run,path=/v1/studio/sdk/versions.warmup_get,method=POST,connector=studio,view=warmup_run\" routeName:\"warmup_run\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.sdk.versions.warmup_get\\\",\\\"description\\\":\\\"Read one authorized Datly Studio cache warmup run\\\"}]\" caseFormat:\"lc\""
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
