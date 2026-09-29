package store_candidate

// Input is the generated input scaffold for definition.
type Input struct {
	NamespaceId         string    `parameter:"NamespaceId,kind=query,in=namespaceId,dataType=string,required=false"`
	CandidateGeneration int64     `parameter:"CandidateGeneration,kind=query,in=candidateGeneration,dataType=int64,required=true"`
	Has                 *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	NamespaceId         bool
	CandidateGeneration bool
}
