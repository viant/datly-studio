package store_insert

// Input is the generated input scaffold for event.
type Input struct {
	Events []*StoredEvent `parameter:"Events,kind=body,in=data,dataType=[]*StoredEvent" view:"event,type=StoredEvent,entityHooks=PublicationEventRules,table=report_publication_events" sql:"uri=studio_report_publication_events_store_insert_event:sql/read.sql"`
	Has    *InputHas      `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Events bool
}
