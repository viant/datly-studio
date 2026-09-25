package store_run_access

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for access.
type AccessComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"access,path=/_studio/report-run-access,method=GET,connector=studio,view=access\" routeName:\"access\" caseFormat:\"lc\""
}

// AccessDatlyType keeps the public component type linked for blank-import discovery.
func AccessDatlyType() reflect.Type { return reflect.TypeOf((*AccessComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var AccessDatly = new(AccessComponent)
var AccessDatlyLinkedType = AccessDatlyType()

func (AccessComponent) EmbedFS() *embed.FS {
	return &AccessDatlyResources
}

func (AccessComponent) EmbedNamespace() string {
	return AccessDatlyResourceNamespace
}
