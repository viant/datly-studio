package store_stage

// Input is the generated input scaffold for publication.
type Input struct {
	NextGeneration                 int64                          `parameter:"NextGeneration,kind=query,in=nextGeneration,dataType=int64,required=true"`
	Publications                   []*StoredPublication           `parameter:"Publications,kind=body,in=data,dataType=[]*StoredPublication" view:"publication,type=StoredPublication,entityHooks=PublicationStageRules,table=report_publications" sql:"uri=studio_report_publications_store_stage_publication:sql/read.sql"`
	PublicationKeys                []PublicationKeysRow           `parameter:"PublicationKeys,kind=param,in=Publications,cardinality=Many" codec:"structql,'uri=studio_report_publications_store_stage_publication:sql/publication_keys.sql'"`
	CurrentPublication             []*CurrentPublicationView      `parameter:"CurrentPublication,kind=view,in=CurrentPublication,cardinality=Many" view:"CurrentPublication,table=report_publications" sql:"uri=studio_report_publications_store_stage_publication:sql/current_publication.sql"`
	_publicationHandlerReadIndexes *PublicationHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                            *InputHas                      `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NextGeneration     bool
	Publications       bool
	PublicationKeys    bool
	CurrentPublication bool
}
