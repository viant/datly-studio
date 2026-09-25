package store_runtime_catalog

// Input is the generated input scaffold for publication.
type Input struct {
	Status string    `parameter:"Status,kind=query,in=status,dataType=string,required=true" predicate:"equal,group=1,c,publication_status"`
	Live   bool      `parameter:"Live,kind=query,in=live,dataType=bool,required=true" predicate:"equal,group=1,c,report_live"`
	Has    *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Status bool
	Live   bool
}
