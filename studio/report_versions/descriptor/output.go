package descriptor

import (
	json "encoding/json"
)

// VersionDescriptorOutput is the generated output scaffold for version.
type VersionDescriptorOutput struct {
	Item      *VersionDescriptor `parameter:"Item,kind=output,in=view,dataType=*VersionDescriptor" json:"-" view:"version,type=VersionDescriptor,table=component_versions,limit=1" sql:"uri=studio_report_versions_descriptor_version:sql/version.sql"`
	Component json.RawMessage    `parameter:"Component,kind=output,in=body,dataType=json.RawMessage" json:"component"`
	Types     json.RawMessage    `parameter:"Types,kind=output,in=body,dataType=json.RawMessage" json:"types,omitempty"`
	Resources json.RawMessage    `parameter:"Resources,kind=output,in=body,dataType=json.RawMessage" json:"resources,omitempty"`
}
