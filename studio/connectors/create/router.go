package create

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for connector.
type ConnectorComponent struct {
	Contract xdatly.Component[ConnectorCreateInput, ConnectorCreateOutput] "component:\"connector,path=/v1/studio/sdk/connectors.create,method=POST,connector=studio,view=connector\" routeName:\"connector\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.sdk.connectors.create\\\",\\\"description\\\":\\\"Create a Datly Studio connector for the verified owner\\\"}]\" mutation:\"post\" caseFormat:\"lc\""
}

// ConnectorDatlyType keeps the public component type linked for blank-import discovery.
func ConnectorDatlyType() reflect.Type { return reflect.TypeOf((*ConnectorComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var ConnectorDatly = new(ConnectorComponent)
var ConnectorDatlyLinkedType = ConnectorDatlyType()

func (ConnectorComponent) EmbedFS() *embed.FS {
	return &ConnectorDatlyResources
}

func (ConnectorComponent) EmbedNamespace() string {
	return ConnectorDatlyResourceNamespace
}
