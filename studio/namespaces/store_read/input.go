package store_read

// Input is the generated input scaffold for namespace.
type Input struct {
	Name       string    `parameter:"Name,kind=query,in=name,dataType=string,required=false"`
	OwnerId    string    `parameter:"OwnerId,kind=query,in=ownerId,dataType=string,required=false"`
	Query      string    `parameter:"Query,kind=query,in=q,dataType=string,required=false"`
	Status     string    `parameter:"Status,kind=query,in=status,dataType=string,required=false"`
	Subject    string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=false"`
	Scoped     bool      `parameter:"Scoped,kind=query,in=scoped,dataType=bool,required=true"`
	PageLimit  int       `parameter:"PageLimit,kind=query,in=pageLimit,dataType=int,required=true"`
	PageOffset int       `parameter:"PageOffset,kind=query,in=pageOffset,dataType=int,required=true"`
	Has        *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Name       bool
	OwnerId    bool
	Query      bool
	Status     bool
	Subject    bool
	Scoped     bool
	PageLimit  bool
	PageOffset bool
}
