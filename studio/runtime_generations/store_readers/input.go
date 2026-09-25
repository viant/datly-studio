package store_readers

// Input is the generated input scaffold for reader.
type Input struct {
	Generation int64     `parameter:"Generation,kind=query,in=generation,dataType=int64,required=true"`
	Subject    string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true"`
	Scoped     bool      `parameter:"Scoped,kind=query,in=scoped,dataType=bool,required=true" predicate:"handler,group=3,github.com/viant/datly-studio/studio/runtime_generations/catalogpredicate.ReaderScope"`
	Has        *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Generation bool
	Subject    bool
	Scoped     bool
}
