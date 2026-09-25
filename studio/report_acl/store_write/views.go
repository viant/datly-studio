package store_write

// StoredACL is generated canonical view metadata for acl.
type StoredACL struct {
	ReportId     *string       `sqlx:"report_id,primaryKey" validate:"required"`
	SubjectType  *string       `sqlx:"subject_type,primaryKey" validate:"required"`
	SubjectId    *string       `sqlx:"subject_id,primaryKey" validate:"required"`
	ShouldDelete bool          `sqlx:"-" writer:"delete"`
	Etag         *int          `writer:"concurrency" sqlx:"etag"`
	CanView      *int          `sqlx:"can_view"`
	CanRun       *int          `sqlx:"can_run"`
	CanEdit      *int          `sqlx:"can_edit"`
	CanPublish   *int          `sqlx:"can_publish"`
	CanUseDql    *int          `sqlx:"can_use_dql"`
	Has          *StoredACLHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredACLHas"`
}

type StoredACLHas struct {
	ReportId     bool
	SubjectType  bool
	SubjectId    bool
	ShouldDelete bool
	Etag         bool
	CanView      bool
	CanRun       bool
	CanEdit      bool
	CanPublish   bool
	CanUseDql    bool
}

// CurrentAclView is generated canonical view metadata for acl.
type CurrentAclView struct {
	ReportId    *string `sqlx:"report_id,primaryKey" validate:"required"`
	SubjectType *string `sqlx:"subject_type,primaryKey" validate:"required"`
	SubjectId   *string `sqlx:"subject_id,primaryKey" validate:"required"`
	Etag        *int    `sqlx:"etag"`
	CanView     *int    `sqlx:"can_view"`
	CanRun      *int    `sqlx:"can_run"`
	CanEdit     *int    `sqlx:"can_edit"`
	CanPublish  *int    `sqlx:"can_publish"`
	CanUseDql   *int    `sqlx:"can_use_dql"`
}

type AclKeysRow struct {
	ReportId    *string `sqlx:"report_id"`
	SubjectType *string `sqlx:"subject_type"`
	SubjectId   *string `sqlx:"subject_id"`
}
