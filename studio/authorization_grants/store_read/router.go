package store_read

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for grant.
type GrantComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"grant,path=/_studio/authorization-grant-store/read,method=GET,connector=studio,view=grant\" routeName:\"grant\" caseFormat:\"lc\""
}

// GrantDatlyType keeps the public component type linked for blank-import discovery.
func GrantDatlyType() reflect.Type { return reflect.TypeOf((*GrantComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var GrantDatly = new(GrantComponent)
var GrantDatlyLinkedType = GrantDatlyType()

func (GrantComponent) EmbedFS() *embed.FS {
	return &GrantDatlyResources
}

func (GrantComponent) EmbedNamespace() string {
	return GrantDatlyResourceNamespace
}
