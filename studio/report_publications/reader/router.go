package reader

import (
	"embed"

	"reflect"

	xdatly "github.com/viant/xdatly"
)

func init() {}

// Component is the generated component scaffold for publication.
type PublicationComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"publication,path=/v1/studio/reports/{reportId}/publication,method=GET,connector=studio,view=publication\" routeName:\"publication\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_publications.read\\\",\\\"description\\\":\\\"Read report publication state\\\"}]\" caseFormat:\"lc\""
}

// PublicationDatlyType keeps the public component type linked for blank-import discovery.
func PublicationDatlyType() reflect.Type { return reflect.TypeOf((*PublicationComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var PublicationDatly = new(PublicationComponent)
var PublicationDatlyLinkedType = PublicationDatlyType()

func (PublicationComponent) EmbedFS() *embed.FS {
	return &PublicationDatlyResources
}

func (PublicationComponent) EmbedNamespace() string {
	return PublicationDatlyResourceNamespace
}
