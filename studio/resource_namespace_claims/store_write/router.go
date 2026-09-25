package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for claim.
type ClaimComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"claim,path=/_studio/resource-namespace-claim-store/write,method=PATCH,connector=studio,view=claim\" routeName:\"claim\" mutation:\"patch\" caseFormat:\"lc\""
}

// ClaimDatlyType keeps the public component type linked for blank-import discovery.
func ClaimDatlyType() reflect.Type { return reflect.TypeOf((*ClaimComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ClaimDatly = new(ClaimComponent)
var ClaimDatlyLinkedType = ClaimDatlyType()

func (ClaimComponent) EmbedFS() *embed.FS {
	return &ClaimDatlyResources
}

func (ClaimComponent) EmbedNamespace() string {
	return ClaimDatlyResourceNamespace
}
