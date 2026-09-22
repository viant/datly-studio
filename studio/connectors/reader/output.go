package reader

// Output is the generated output scaffold for connector.
type Output struct {
	Connectors []*Connector `parameter:"Connectors,kind=output,in=view,dataType=[]*Connector" view:"connector,type=Connector,table=connectors,limit=100,selectorProjection=true,selectorOrderBy=true,selectorLimit=true,selectorOffset=true,selectorOrderable={name,status,driver,owner_id,updated_at}" sql:"uri=studio_connectors_reader_connector:sql/connector.sql"`
}
