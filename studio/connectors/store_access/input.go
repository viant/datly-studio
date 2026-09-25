package store_access

// Input is the generated input scaffold for connector.
type Input struct {
	Name       string    `parameter:"Name,kind=query,in=name,dataType=string,required=true"`
	Subject    string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true"`
	Permission string    `parameter:"Permission,kind=query,in=permission,dataType=string,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/connectors/accesspredicate.ConnectorAccess"`
	Has        *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Name       bool
	Subject    bool
	Permission bool
}
