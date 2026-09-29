package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for claim.
type ClaimComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"claim,path=/_studio/resource-namespace-claim-store/write,method=PATCH,connector=studio,view=claim,internal=true\" routeName:\"claim\" mutation:\"patch\" caseFormat:\"lc\""
}

// ClaimDatlyType returns the public component type.
func ClaimDatlyType() reflect.Type { return reflect.TypeOf((*ClaimComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var ClaimDatly = new(ClaimComponent)
var _datlyReachableClaimComponent = reflect.TypeFor[ClaimComponent]()

func (ClaimComponent) EmbedFS() *embed.FS {
	return &ClaimDatlyResources
}

func (ClaimComponent) EmbedNamespace() string {
	return ClaimDatlyResourceNamespace
}
