package store_draft_pointer

func (input *Input) SetReports(value []*DraftPointer) {
	if input == nil {
		return
	}
	input.Reports = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.Reports = true
}

func (input *Input) SetReportKeys(value []ReportKeysRow) {
	if input == nil {
		return
	}
	input.ReportKeys = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ReportKeys = true
}

func (input *Input) SetCurrentReport(value []*CurrentReportView) {
	if input == nil {
		return
	}
	input.CurrentReport = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.CurrentReport = true
}
