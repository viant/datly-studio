package store_catalog

// Input is the generated input scaffold for report.
type Input struct {
	Id            string    `parameter:"Id,kind=query,in=id,dataType=string,required=false" predicate:"equal,group=2,r,id"`
	Query         string    `parameter:"Query,kind=query,in=q,dataType=string,required=false" predicate:"contains,r,namespace" predicate:"contains,r,slug" predicate:"contains,r,title" predicate:"contains,r,description"`
	Namespace     string    `parameter:"Namespace,kind=query,in=namespace,dataType=string,required=false" predicate:"equal,group=1,r,namespace"`
	Status        string    `parameter:"Status,kind=query,in=status,dataType=string,required=false" predicate:"equal,group=1,r,status"`
	OwnerId       string    `parameter:"OwnerId,kind=query,in=ownerId,dataType=string,required=false" predicate:"equal,group=1,r,owner_id"`
	ConnectorName string    `parameter:"ConnectorName,kind=query,in=connector,dataType=string,required=false" predicate:"equal,group=1,r,default_connector_name"`
	Subject       string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true"`
	Scoped        bool      `parameter:"Scoped,kind=query,in=scoped,dataType=bool,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/reports/catalogpredicate.ReportCatalogRead"`
	Limit         int       `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=report"`
	Offset        int       `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=report"`
	Has           *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Id            bool
	Query         bool
	Namespace     bool
	Status        bool
	OwnerId       bool
	ConnectorName bool
	Subject       bool
	Scoped        bool
	Limit         bool
	Offset        bool
}
