package reader

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for acl.
type AclComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"acl,path=/v1/studio/sdk/acl.list,method=POST,connector=studio,view=acl\" routeName:\"acl\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.sdk.acl.list\\\",\\\"description\\\":\\\"List report ACL entries authorized for the current principal\\\"}]\" caseFormat:\"lc\""
}

// AclDatlyType keeps the public component type linked for blank-import discovery.
func AclDatlyType() reflect.Type { return reflect.TypeOf((*AclComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var AclDatly = new(AclComponent)
var AclDatlyLinkedType = AclDatlyType()

func (AclComponent) EmbedFS() *embed.FS {
	return &AclDatlyResources
}

func (AclComponent) EmbedNamespace() string {
	return AclDatlyResourceNamespace
}
