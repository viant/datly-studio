package store_write

// Input is the generated input scaffold for acl.
type Input struct {
	Access                 []*StoredACL           `parameter:"Access,kind=body,in=data,dataType=[]*StoredACL" view:"acl,type=StoredACL,entityHooks=ACLStoreRules,table=report_acl" sql:"uri=studio_report_acl_store_write_acl:sql/patch.sql"`
	AclKeys                []AclKeysRow           `parameter:"AclKeys,kind=param,in=Access,cardinality=Many" codec:"structql,'uri=studio_report_acl_store_write_acl:sql/acl_keys.sql'"`
	CurrentAcl             []*CurrentAclView      `parameter:"CurrentAcl,kind=view,in=CurrentAcl,cardinality=Many" view:"CurrentAcl,table=report_acl" sql:"uri=studio_report_acl_store_write_acl:sql/current_acl.sql"`
	_aclHandlerReadIndexes *AclHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                    *InputHas              `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Access     bool
	AclKeys    bool
	CurrentAcl bool
}
