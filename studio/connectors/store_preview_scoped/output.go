package store_preview_scoped

// Output is the generated output scaffold for connector.
type Output struct {
	Connectors []*PreviewConnector `parameter:"Connectors,kind=output,in=view,dataType=[]*PreviewConnector" view:"connector,type=PreviewConnector,table=connectors,selectorNoLimit=true" sql:"uri=studio_connectors_store_preview_scoped_connector:sql/read.sql"`
}
