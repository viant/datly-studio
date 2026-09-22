package writer

import (
	jwt "github.com/viant/scy/auth/jwt"
)

// Input is the generated input scaffold for acl.
type Input struct {
	Jwt *jwt.Claims `parameter:"Jwt,kind=header,in=Authorization,dataType=string,errorCode=401,required=true" codec:"JwtClaim"`
	Access []*ReportACL `parameter:"Access,kind=body,in=data,dataType=[]*ReportACL" view:"acl,type=ReportACL,entityHooks=ACLRules,table=report_acl" sql:"uri=studio_report_acl_writer_acl:sql/acl.sql"`
	AclKeys []AclKeysRow `parameter:"AclKeys,kind=param,in=Access,cardinality=Many" codec:"structql,'uri=studio_report_acl_writer_acl:sql/acl_keys.sql'"`
	CurrentAcl []*CurrentAclView `parameter:"CurrentAcl,kind=view,in=CurrentAcl,cardinality=Many" view:"CurrentAcl,table=report_acl" sql:"uri=studio_report_acl_writer_acl:sql/current_acl.sql"`
	_aclHandlerReadIndexes *AclHandlerReadIndexes `json:"-" sqlx:"-"`
	Has *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Jwt bool
	Access bool
	AclKeys bool
	CurrentAcl bool
}
