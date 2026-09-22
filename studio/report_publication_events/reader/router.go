package reader

import (
	"embed"
	"reflect"

	xdatly "github.com/viant/xdatly"
)

type PublicationEventComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"event,path=/v1/studio/publication-events,method=GET,connector=studio,view=event\" routeName:\"event\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.report_publication_events.read\\\",\\\"description\\\":\\\"Read owner-scoped publication lifecycle evidence\\\"}]\" caseFormat:\"lc\""
}

func PublicationEventDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationEventComponent)(nil)).Elem()
}

var PublicationEventDatly = new(PublicationEventComponent)
var PublicationEventDatlyLinkedType = PublicationEventDatlyType()

func (PublicationEventComponent) EmbedFS() *embed.FS { return &PublicationEventDatlyResources }
func (PublicationEventComponent) EmbedNamespace() string {
	return PublicationEventDatlyResourceNamespace
}
