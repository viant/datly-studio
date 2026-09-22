package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for parameter.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Parameters []*ReportParameter `parameter:"Parameters,kind=body,in=data,dataType=[]*ReportParameter" view:"parameter,type=ReportParameter,entityHooks=ParameterRules,table=report_parameters" sql:"uri=studio_report_parameters_writer_parameter:sql/parameter.sql"`
	ParameterKeys []ParameterKeysRow `parameter:"ParameterKeys,kind=param,in=Parameters,cardinality=Many" codec:"structql,'uri=studio_report_parameters_writer_parameter:sql/parameter_keys.sql'"`
	CurrentParameter []*CurrentParameterView `parameter:"CurrentParameter,kind=view,in=CurrentParameter,cardinality=Many" view:"CurrentParameter,table=report_parameters" sql:"uri=studio_report_parameters_writer_parameter:sql/current_parameter.sql"`
	CurrentPredicate []*CurrentPredicateView `parameter:"CurrentPredicate,kind=view,in=CurrentPredicate,cardinality=Many" view:"CurrentPredicate,table=report_predicates" sql:"uri=studio_report_parameters_writer_parameter:sql/current_predicate.sql"`
	_parameterHandlerReadIndexes *ParameterHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Parameters bool
	ParameterKeys bool
	CurrentParameter bool
	CurrentPredicate bool
}
