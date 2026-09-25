package store_insert

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for event.
type EventComponent struct {
	Contract xdatly.Component[Input, Output] "component:\"event,path=/_studio/publication-event-store/insert,method=POST,connector=studio,view=event\" routeName:\"event\" mutation:\"post\" caseFormat:\"lc\""
}

// EventDatlyType keeps the public component type linked for blank-import discovery.
func EventDatlyType() reflect.Type { return reflect.TypeOf((*EventComponent)(nil)).Elem() }

// Datly anchors this package's public component contract.
var EventDatly = new(EventComponent)
var EventDatlyLinkedType = EventDatlyType()

func (EventComponent) EmbedFS() *embed.FS {
	return &EventDatlyResources
}

func (EventComponent) EmbedNamespace() string {
	return EventDatlyResourceNamespace
}
