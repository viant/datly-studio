package store_write

// Input is the generated input scaffold for claim.
type Input struct {
	Operation                string                   `parameter:"Operation,kind=query,in=operation,dataType=string,required=true"`
	Claims                   []*StoredClaim           `parameter:"Claims,kind=body,in=data,dataType=[]*StoredClaim" view:"claim,type=StoredClaim,entityHooks=NamespaceClaimRules,table=resource_namespace_claims" sql:"uri=studio_resource_namespace_claims_store_write_claim:sql/read.sql"`
	ClaimKeys                []ClaimKeysRow           `parameter:"ClaimKeys,kind=param,in=Claims,cardinality=Many" codec:"structql,'uri=studio_resource_namespace_claims_store_write_claim:sql/claim_keys.sql'"`
	CurrentClaim             []*CurrentClaimView      `parameter:"CurrentClaim,kind=view,in=CurrentClaim,cardinality=Many" view:"CurrentClaim,table=resource_namespace_claims" sql:"uri=studio_resource_namespace_claims_store_write_claim:sql/current_claim.sql"`
	_claimHandlerReadIndexes *ClaimHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                      *InputHas                `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Operation    bool
	Claims       bool
	ClaimKeys    bool
	CurrentClaim bool
}
