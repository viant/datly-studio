package store_candidate

func (input *Input) SetCandidateGeneration(value int64) {
	if input == nil {
		return
	}
	input.CandidateGeneration = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CandidateGeneration = true
}
