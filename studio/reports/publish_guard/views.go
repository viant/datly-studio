package publish_guard

// Report is generated canonical view metadata for report.
type Report struct {
	Id        string `sqlx:"id,required=true,primaryKey=true"`
	OwnerId   string `sqlx:"owner_id,refTable=namespaces,refColumn=owner_id,required=true"`
	Namespace string `sqlx:"namespace,refTable=namespaces,refColumn=name,required=true"`
}
