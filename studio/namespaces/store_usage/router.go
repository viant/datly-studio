package store_usage

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for usage.
type UsageComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"usage,path=/_studio/namespace-store/usage,method=GET,connector=studio,view=usage\" routeName:\"usage\" caseFormat:\"lc\""
}

// UsageDatlyType keeps the public component type linked for blank-import discovery.
func UsageDatlyType() reflect.Type { return reflect.TypeOf((*UsageComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var UsageDatly = new(UsageComponent)
var UsageDatlyLinkedType = UsageDatlyType()

func (UsageComponent) EmbedFS() *embed.FS {
	return &UsageDatlyResources
}

func (UsageComponent) EmbedNamespace() string {
	return UsageDatlyResourceNamespace
}
