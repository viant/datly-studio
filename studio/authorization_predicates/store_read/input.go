package store_read

// Input is the generated input scaffold for authorization_predicate.
type Input struct {
	Name          string    `parameter:"Name,kind=query,in=name,dataType=string,required=false" predicate:"equal,p,name"`
	SearchPattern string    `parameter:"SearchPattern,kind=query,in=searchPattern,dataType=string,required=false"`
	Status        string    `parameter:"Status,kind=query,in=status,dataType=string,required=false"`
	OrderBy       string    `parameter:"OrderBy,kind=query,in=orderBy,dataType=string,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	Limit         int       `parameter:"Limit,kind=query,in=limit,dataType=int,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	Offset        int       `parameter:"Offset,kind=query,in=offset,dataType=int,required=false,cacheable=false" querySelector:"view=authorization_predicate"`
	Has           *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Name          bool
	SearchPattern bool
	Status        bool
	OrderBy       bool
	Limit         bool
	Offset        bool
}
