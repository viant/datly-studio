package store_status

// Output is the generated output scaffold for connector.
type Output struct {
	Data []*StoredConnector `parameter:"Data,kind=output,in=body,dataType=[]*StoredConnector"`
}
