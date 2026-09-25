package store_insert

// Input is the generated input scaffold for publication.
type Input struct {
	Publications []*StoredPublication `parameter:"Publications,kind=body,in=data,dataType=[]*StoredPublication" view:"publication,type=StoredPublication,entityHooks=PublicationInsertRules,table=report_publications" sql:"uri=studio_report_publications_store_insert_publication:sql/read.sql"`
	Has          *InputHas            `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Publications bool
}
