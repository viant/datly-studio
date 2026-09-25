package get

import (
	json "encoding/json"
	time "time"
)

// ConnectorGetOutput is the generated output scaffold for connector.
type ConnectorGetOutput struct {
	Item              *Connector      `parameter:"Item,kind=output,in=view,dataType=*Connector" json:"-" view:"connector,type=Connector,table=connectors,limit=1" sql:"uri=studio_connectors_get_connector:sql/connector.sql"`
	ResponseName      string          `parameter:"ResponseName,kind=output,in=body,dataType=string" json:"name"`
	Driver            string          `parameter:"Driver,kind=output,in=body,dataType=string" json:"driver"`
	DsnConfigured     bool            `parameter:"DsnConfigured,kind=output,in=body,dataType=bool" json:"dsnConfigured"`
	SecretConfigured  bool            `parameter:"SecretConfigured,kind=output,in=body,dataType=bool" json:"secretConfigured"`
	Description       string          `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	OwnerId           string          `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	Status            string          `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	Options           json.RawMessage `parameter:"Options,kind=output,in=body,dataType=json.RawMessage" json:"options,omitempty"`
	LastTestStatus    string          `parameter:"LastTestStatus,kind=output,in=body,dataType=string" json:"lastTestStatus,omitempty"`
	LastTestErrorCode string          `parameter:"LastTestErrorCode,kind=output,in=body,dataType=string" json:"lastTestErrorCode,omitempty"`
	LastTestedAt      *time.Time      `parameter:"LastTestedAt,kind=output,in=body,dataType=*time.Time" json:"lastTestedAt,omitempty"`
	Etag              int64           `parameter:"Etag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt         time.Time       `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt         time.Time       `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}
