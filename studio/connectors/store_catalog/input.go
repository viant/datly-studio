package store_catalog

// Input is the generated input scaffold for connector.
type Input struct {
	Name       string    `parameter:"Name,kind=query,in=name,dataType=string,required=true"`
	Query      string    `parameter:"Query,kind=query,in=q,dataType=string,required=true"`
	Status     string    `parameter:"Status,kind=query,in=status,dataType=string,required=true"`
	OwnerId    string    `parameter:"OwnerId,kind=query,in=ownerId,dataType=string,required=true"`
	Driver     string    `parameter:"Driver,kind=query,in=driver,dataType=string,required=true"`
	Subject    string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true"`
	Scoped     bool      `parameter:"Scoped,kind=query,in=scoped,dataType=bool,required=true"`
	PageLimit  int       `parameter:"PageLimit,kind=query,in=pageLimit,dataType=int,required=true"`
	PageOffset int       `parameter:"PageOffset,kind=query,in=pageOffset,dataType=int,required=true"`
	Has        *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Name       bool
	Query      bool
	Status     bool
	OwnerId    bool
	Driver     bool
	Subject    bool
	Scoped     bool
	PageLimit  bool
	PageOffset bool
}
