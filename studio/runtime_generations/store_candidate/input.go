package store_candidate

// Input is the generated input scaffold for definition.
type Input struct {
	CandidateGeneration int64     `parameter:"CandidateGeneration,kind=query,in=candidateGeneration,dataType=int64,required=true"`
	Has                 *InputHas `setMarker:"true" typeName:"InputHas" json:"-" sqlx:"-"`
}

type InputHas struct {
	CandidateGeneration bool
}
