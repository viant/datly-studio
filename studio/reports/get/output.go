package get

import (
	time "time"
)

// ReportGetOutput is the generated output scaffold for report.
type ReportGetOutput struct {
	Item                 *Report   `parameter:"Item,kind=output,in=view,dataType=*Report" json:"-" view:"report,type=Report,table=reports,limit=1" sql:"uri=studio_reports_get_report:sql/report.sql"`
	ResponseId           string    `parameter:"ResponseId,kind=output,in=body,dataType=string" json:"id"`
	Namespace            string    `parameter:"Namespace,kind=output,in=body,dataType=string" json:"namespace"`
	Slug                 string    `parameter:"Slug,kind=output,in=body,dataType=string" json:"slug"`
	Title                string    `parameter:"Title,kind=output,in=body,dataType=string" json:"title"`
	Description          string    `parameter:"Description,kind=output,in=body,dataType=string" json:"description,omitempty"`
	OwnerId              string    `parameter:"OwnerId,kind=output,in=body,dataType=string" json:"ownerId"`
	OwnerPackage         string    `parameter:"OwnerPackage,kind=output,in=body,dataType=string" json:"ownerPackage"`
	Status               string    `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	DefaultConnectorName string    `parameter:"DefaultConnectorName,kind=output,in=body,dataType=string" json:"defaultConnectorName"`
	ComponentScope       string    `parameter:"ComponentScope,kind=output,in=body,dataType=string" json:"componentScope"`
	ComponentName        string    `parameter:"ComponentName,kind=output,in=body,dataType=string" json:"componentName"`
	CurrentDraftVersion  *int      `parameter:"CurrentDraftVersion,kind=output,in=body,dataType=*int" json:"currentDraftVersion,omitempty"`
	Etag                 int64     `parameter:"Etag,kind=output,in=body,dataType=int64" json:"etag"`
	CreatedAt            time.Time `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	UpdatedAt            time.Time `parameter:"UpdatedAt,kind=output,in=body,dataType=time.Time" json:"updatedAt"`
}
