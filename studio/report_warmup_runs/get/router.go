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
