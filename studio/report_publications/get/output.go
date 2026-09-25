package get

import (
	time "time"
)

// PublicationGetOutput is the generated output scaffold for publication.
type PublicationGetOutput struct {
	Item              *Publication `parameter:"Item,kind=output,in=view,dataType=*Publication" json:"-" view:"publication,type=Publication,table=report_publications,limit=1" sql:"uri=studio_report_publications_get_publication:sql/publication.sql"`
	ResponseReportId  string       `parameter:"ResponseReportId,kind=output,in=body,dataType=string" json:"reportId"`
	ActiveVersionNo   int          `parameter:"ActiveVersionNo,kind=output,in=body,dataType=int" json:"activeVersionNo"`
	DesiredVersionNo  *int         `parameter:"DesiredVersionNo,kind=output,in=body,dataType=*int" json:"desiredVersionNo,omitempty"`
	DesiredGeneration int64        `parameter:"DesiredGeneration,kind=output,in=body,dataType=int64" json:"desiredGeneration"`
	ActiveGeneration  *int64       `parameter:"ActiveGeneration,kind=output,in=body,dataType=*int64" json:"activeGeneration,omitempty"`
	Status            string       `parameter:"Status,kind=output,in=body,dataType=string" json:"status"`
	RuntimeRevision   string       `parameter:"RuntimeRevision,kind=output,in=body,dataType=string" json:"runtimeRevision,omitempty"`
	SpecHash          string       `parameter:"SpecHash,kind=output,in=body,dataType=string" json:"specHash,omitempty"`
	PublishedAt       *time.Time   `parameter:"PublishedAt,kind=output,in=body,dataType=*time.Time" json:"publishedAt,omitempty"`
}
