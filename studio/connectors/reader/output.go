package reader

// Output is the generated output scaffold for connector.
type Output struct {
	Items      []*Connector `parameter:"Items,kind=output,in=view,dataType=[]*Connector" view:"connector,type=Connector,table=connectors,limit=500,selectorLimit=true,selectorOffset=true" sql:"uri=studio_connectors_reader_connector:sql/connector.sql"`
	PageLimit  int          `parameter:"PageLimit,kind=output,in=body,dataType=int" json:"limit"`
	PageOffset int          `parameter:"PageOffset,kind=output,in=body,dataType=int" json:"offset"`
}
