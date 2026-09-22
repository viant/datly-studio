package reader

type Output struct {
	Events []*PublicationEvent `parameter:"Events,kind=output,in=view,dataType=[]*PublicationEvent" view:"event,type=PublicationEvent,table=report_publication_events,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={occurred_at,operation,status,version_no,generation_no,requested_by}" sql:"uri=studio_report_publication_events_reader_event:sql/read.sql"`
}
