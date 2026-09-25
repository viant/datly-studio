package list

// PublicationEventsListOutput is the generated output scaffold for event.
type PublicationEventsListOutput struct {
	Items      []*PublicationEvent `parameter:"Items,kind=output,in=view,dataType=[]*PublicationEvent" view:"event,type=PublicationEvent,selectorNoLimit=true" sql:"uri=studio_report_publication_events_list_event:sql/event.sql"`
	PageLimit  int                 `parameter:"PageLimit,kind=output,in=body,dataType=int" json:"limit"`
	PageOffset int                 `parameter:"PageOffset,kind=output,in=body,dataType=int" json:"offset"`
}
