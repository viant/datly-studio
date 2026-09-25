package store_capabilities

// ReportCapability is generated canonical view metadata for capability.
type ReportCapability struct {
	ReportId   string `sqlx:"report_id"`
	OwnerId    string `sqlx:"owner_id"`
	CanView    bool   `sqlx:"can_view"`
	CanRun     bool   `sqlx:"can_run"`
	CanEdit    bool   `sqlx:"can_edit"`
	CanPublish bool   `sqlx:"can_publish"`
	CanUseDql  bool   `sqlx:"can_use_dql"`
}
