package store_preview_scoped

// Input is the generated input scaffold for connector.
type Input struct {
	Subject string    `parameter:"Subject,kind=query,in=subject,dataType=string,required=true"`
	Has     *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	Subject bool
}
