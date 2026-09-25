package get

import (
	time "time"
)

// NamespaceGetOutput is the generated output scaffold for namespace.
type NamespaceGetOutput struct {
	Item         *NamespaceRecord `parameter:"Item,kind=output,in=view,dataType=*NamespaceRecord" json:"-" view:"namespace,type=NamespaceRecord,table=namespaces,limit=1" sql:"uri=studio_namespaces_get_namespace:sql/namespace.sql"`
	OwnerId      string           `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	ResponseName string           `parameter:"ResponseName,kind=output,in=body,dataType=string" json:"name"`
	Title        string           `parameter:"Title,kind=output,in=body,dataType=string" json:"title"`
	Description  string           `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	Status       string           `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	Etag         int64            `parameter:"Etag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt    time.Time        `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt    time.Time        `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}
