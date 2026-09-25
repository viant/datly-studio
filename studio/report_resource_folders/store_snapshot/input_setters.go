package store_snapshot

func (input *Input) SetReportId(value string) {
	if input == nil {
		return
	}
	input.ReportId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.ReportId = true
}

func (input *Input) SetVersionNo(value int) {
	if input == nil {
		return
	}
	input.VersionNo = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.VersionNo = true
}

func (input *Input) SetFolderId(value string) {
	if input == nil {
		return
	}
	input.FolderId = value
	if input.Has == nil {
		input.Has = &InputHas{}
	}
	input.Has.FolderId = true
}
