package store_usage

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for usage.
type UsageComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"usage,path=/_studio/resource-namespace-store/usage,method=GET,connector=studio,view=usage,internal=true\" routeName:\"usage\" caseFormat:\"lc\""
}

// UsageDatlyType returns the public component type.
func UsageDatlyType() reflect.Type { return reflect.TypeOf((*UsageComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var UsageDatly = new(UsageComponent)
var _datlyReachableUsageComponent = reflect.TypeFor[UsageComponent]()

func (UsageComponent) EmbedFS() *embed.FS {
	return &UsageDatlyResources
}

func (UsageComponent) EmbedNamespace() string {
	return UsageDatlyResourceNamespace
}
