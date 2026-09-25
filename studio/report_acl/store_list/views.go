package store_list

// StoredACL is generated canonical view metadata for acl.
type StoredACL struct {
	ReportId    string `sqlx:"report_id"`
	SubjectType string `sqlx:"subject_type"`
	SubjectId   string `sqlx:"subject_id"`
	CanView     bool   `sqlx:"can_view"`
	CanRun      bool   `sqlx:"can_run"`
	CanEdit     bool   `sqlx:"can_edit"`
	CanPublish  bool   `sqlx:"can_publish"`
	CanUseDql   bool   `sqlx:"can_use_dql"`
	Etag        int64  `sqlx:"etag"`
}
