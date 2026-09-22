package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for publication.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Publications []*ReportPublication `parameter:"Publications,kind=body,in=data,dataType=[]*ReportPublication" view:"publication,type=ReportPublication,entityHooks=PublicationRules,table=report_publications" sql:"uri=studio_report_publications_writer_publication:sql/patch.sql"`
	PublicationKeys []PublicationKeysRow `parameter:"PublicationKeys,kind=param,in=Publications,cardinality=Many" codec:"structql,'uri=studio_report_publications_writer_publication:sql/publication_keys.sql'"`
	CurrentPublication []*CurrentPublicationView `parameter:"CurrentPublication,kind=view,in=CurrentPublication,cardinality=Many" view:"CurrentPublication,table=report_publications" sql:"uri=studio_report_publications_writer_publication:sql/current_publication.sql"`
	_publicationHandlerReadIndexes *PublicationHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Publications bool
	PublicationKeys bool
	CurrentPublication bool
}
