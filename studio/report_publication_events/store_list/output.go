package store_list

// Output is the generated output scaffold for event.
type Output struct {
	Events []*StoredEvent `parameter:"Events,kind=output,in=view,dataType=[]*StoredEvent" view:"event,type=StoredEvent,selectorNoLimit=true" sql:"uri=studio_report_publication_events_store_list_event:sql/read.sql"`
}
