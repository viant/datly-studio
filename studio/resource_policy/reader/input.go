package reader

// Input is the generated input scaffold for policy.
type Input struct {
	TenantId        string    `parameter:"TenantId,kind=query,in=tenantId,dataType=string,required=true" predicate:"equal,h,tenant_id"`
	ResourceKind    string    `parameter:"ResourceKind,kind=query,in=resourceKind,dataType=string,required=true" predicate:"equal,h,resource_kind"`
	ResourceId      string    `parameter:"ResourceId,kind=query,in=resourceId,dataType=string,required=true" predicate:"equal,h,resource_id"`
	ResourceVersion string    `parameter:"ResourceVersion,kind=query,in=resourceVersion,dataType=string,required=true" predicate:"equal,h,resource_version"`
	Has             *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	TenantId        bool
	ResourceKind    bool
	ResourceId      bool
	ResourceVersion bool
}
