package store_insert

func (input *Input) SetReports(value []*StoredReport) {
	if input == nil {
		return
	}
	input.Reports = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Reports = true
}
