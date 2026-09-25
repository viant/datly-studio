package store_repoint

// Input is the generated input scaffold for publication.
type Input struct {
	ExcludeReportId                string                         `parameter:"ExcludeReportId,kind=query,in=excludeReportId,dataType=string,required=true"`
	Generation                     int64                          `parameter:"Generation,kind=query,in=generation,dataType=int64,required=true"`
	Publications                   []*StoredPublication           `parameter:"Publications,kind=body,in=data,dataType=[]*StoredPublication" view:"publication,type=StoredPublication,entityHooks=PublicationRepointRules,table=report_publications" sql:"uri=studio_report_publications_store_repoint_publication:sql/read.sql"`
	PublicationKeys                []PublicationKeysRow           `parameter:"PublicationKeys,kind=param,in=Publications,cardinality=Many" codec:"structql,'uri=studio_report_publications_store_repoint_publication:sql/publication_keys.sql'"`
	CurrentPublication             []*CurrentPublicationView      `parameter:"CurrentPublication,kind=view,in=CurrentPublication,cardinality=Many" view:"CurrentPublication,table=report_publications" sql:"uri=studio_report_publications_store_repoint_publication:sql/current_publication.sql"`
	_publicationHandlerReadIndexes *PublicationHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                            *InputHas                      `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	ExcludeReportId    bool
	Generation         bool
	Publications       bool
	PublicationKeys    bool
	CurrentPublication bool
}
