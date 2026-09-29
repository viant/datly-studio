package list

import (
	embed "embed"
	xdatly "github.com/viant/xdatly"
	reflect "reflect"
)

func init() {}

// Component is the generated component scaffold for event.
type EventComponent struct {
	Contract xdatly.Component[PublicationEventsListInput, PublicationEventsListOutput] "component:\"event,path=/v1/studio/sdk/publications.events.list,method=POST,connector=studio,view=event\" routeName:\"event\" mcp:\"[{\\\"kind\\\":\\\"tool\\\",\\\"name\\\":\\\"studio.sdk.publications.events.list\\\",\\\"description\\\":\\\"List current-owner publication lifecycle events\\\"}]\" caseFormat:\"lc\""
}

// EventDatlyType returns the public component type.
func EventDatlyType() reflect.Type { return reflect.TypeOf((*EventComponent)(nil)).Elem() }

// The package-level value keeps this real component type reachable for runtime discovery.
var EventDatly = new(EventComponent)
var _datlyReachableEventComponent = reflect.TypeFor[EventComponent]()

func (EventComponent) EmbedFS() *embed.FS {
	return &EventDatlyResources
}

func (EventComponent) EmbedNamespace() string {
	return EventDatlyResourceNamespace
}
