package get

import (
	time "time"
)

// NamespaceGetOutput is the generated output scaffold for namespace.
type NamespaceGetOutput struct {
	Item         *NamespaceRecord `parameter:"Item,kind=output,in=view,dataType=*NamespaceRecord" json:"-" view:"namespace,type=NamespaceRecord,table=namespaces,limit=1" sql:"uri=studio_namespaces_get_namespace:sql/namespace.sql"`
	CanManage    bool             `parameter:"CanManage,kind=output,in=body,dataType=bool" json:"canManage"`
	OwnerId      string           `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	ResponseName string           `parameter:"ResponseName,kind=output,in=body,dataType=string" json:"name"`
	Title        string           `parameter:"Title,kind=output,in=body,dataType=string" json:"title"`
	Description  string           `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	Status       string           `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	Etag         int64            `parameter:"Etag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt    time.Time        `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt    time.Time        `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
	NamespaceID  string           `parameter:"NamespaceID,kind=output,in=body,dataType=string" json:"namespaceId"`
	Visibility   string           `parameter:"Visibility,kind=output,in=body,dataType=string" json:"visibility"`
	AllowedRoles []string         `parameter:"AllowedRoles,kind=output,in=body,dataType=[]string" json:"allowedRoles"`
	MCPEnabled   bool             `parameter:"MCPEnabled,kind=output,in=body,dataType=bool" json:"mcpEnabled"`
	MCPPort      *int             `parameter:"MCPPort,kind=output,in=body,dataType=*int" json:"mcpPort,omitempty"`
}
