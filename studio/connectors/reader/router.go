package reader

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for connector.
type ConnectorComponent struct {
	Contract1 xdatly.Component[Input, Output] "component:\"connector,path=/v1/studio/connectors,method=GET,connector=studio,view=connector\" routeName:\"connector\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.connectors.read\\\",\\\"description\\\":\\\"Read Datly Studio connectors\\\"}]\" caseFormat:\"lc\""
	Contract2 xdatly.Component[Input, Output] "component:\"connector,path=/v1/studio/connectors/{name},method=GET,connector=studio,view=connector\" routeName:\"connector\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.connectors.readByName\\\",\\\"description\\\":\\\"Read Datly Studio connectors\\\"}]\" caseFormat:\"lc\""
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
