package store_write

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for acl.
type AclComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"acl,path=/_studio/report-acl-store,method=PATCH,connector=studio,view=acl,internal=true\" routeName:\"acl\" mutation:\"patch\" caseFormat:\"lc\""
}

// AclDatlyType returns the public component type.
func AclDatlyType() reflect.Type { return reflect.TypeOf((*AclComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var AclDatly = new(AclComponent)
var _datlyReachableAclComponent = reflect.TypeFor[AclComponent]()

func (AclComponent) EmbedFS() *embed.FS {
	return &AclDatlyResources
}

func (AclComponent) EmbedNamespace() string {
	return AclDatlyResourceNamespace
}
