package writer

// Input is the generated input scaffold for policy.
type Input struct {
	Policies                  []*ResourcePolicyHead     `parameter:"Policies,kind=body,in=data,dataType=[]*ResourcePolicyHead" view:"policy,type=ResourcePolicyHead,entityHooks=PolicyRules,table=resource_policy_heads" sql:"uri=studio_resource_policy_writer_policy:sql/policy.sql"`
	PolicyKeys                []PolicyKeysRow           `parameter:"PolicyKeys,kind=param,in=Policies,cardinality=Many" codec:"structql,'uri=studio_resource_policy_writer_policy:sql/policy_keys.sql'"`
	CurrentPolicy             []*CurrentPolicyView      `parameter:"CurrentPolicy,kind=view,in=CurrentPolicy,cardinality=Many" view:"CurrentPolicy,table=resource_policy_heads" sql:"uri=studio_resource_policy_writer_policy:sql/current_policy.sql"`
	CurrentHistory            []*CurrentHistoryView     `parameter:"CurrentHistory,kind=view,in=CurrentHistory,cardinality=Many" view:"CurrentHistory,table=resource_policy_revisions" sql:"uri=studio_resource_policy_writer_policy:sql/current_history.sql"`
	_policyHandlerReadIndexes *PolicyHandlerReadIndexes `json:"-" sqlx:"-"`
	Has                       *InputHas                 `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Policies       bool
	PolicyKeys     bool
	CurrentPolicy  bool
	CurrentHistory bool
}
