package store_read

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for warmup_run.
type WarmupRunComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"warmup_run,path=/_studio/report-warmup-run-store/read,method=GET,connector=studio,view=warmup_run\" routeName:\"warmup_run\" caseFormat:\"lc\""
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
