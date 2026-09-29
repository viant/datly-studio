package presence

// Input is the generated input scaffold for head.
type Input struct {
	Tenant       string    `parameter:"Tenant,kind=query,in=tenant,dataType=string,required=true" predicate:"equal,p,tenant_id"`
	ResourceKind string    `parameter:"ResourceKind,kind=query,in=resourceKind,dataType=string,required=true" predicate:"equal,p,resource_kind"`
	ResourceId   string    `parameter:"ResourceId,kind=query,in=resourceId,dataType=string,required=true" predicate:"equal,p,resource_id"`
	Has          *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Tenant       bool
	ResourceKind bool
	ResourceId   bool
}
