package writer

// ReportACL is generated canonical view metadata for acl.
type ReportACL struct {
	ReportId     *string       `sqlx:"report_id,primaryKey,refTable=reports,refColumn=id,required=true" validate:"required"`
	SubjectType  *string       `sqlx:"subject_type,primaryKey,required=true" validate:"required"`
	SubjectId    *string       `sqlx:"subject_id,primaryKey,required=true" validate:"required"`
	ShouldDelete bool          `sqlx:"-" writer:"delete"`
	Etag         *int          `writer:"concurrency" sqlx:"etag"`
	CanView      *int          `sqlx:"can_view,required=true"`
	CanRun       *int          `sqlx:"can_run,required=true"`
	CanEdit      *int          `sqlx:"can_edit,required=true"`
	CanPublish   *int          `sqlx:"can_publish,required=true"`
	CanUseDql    *int          `sqlx:"can_use_dql,required=true"`
	Has          *ReportACLHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportACLHas"`
}

type ReportACLHas struct {
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
	ReportId    *string `sqlx:"report_id,primaryKey,refTable=reports,refColumn=id,required=true" validate:"required"`
	SubjectType *string `sqlx:"subject_type,primaryKey,required=true" validate:"required"`
	SubjectId   *string `sqlx:"subject_id,primaryKey,required=true" validate:"required"`
	CanView     *int    `sqlx:"can_view,required=true"`
	CanRun      *int    `sqlx:"can_run,required=true"`
	CanEdit     *int    `sqlx:"can_edit,required=true"`
	CanPublish  *int    `sqlx:"can_publish,required=true"`
	CanUseDql   *int    `sqlx:"can_use_dql,required=true"`
	Etag        *int    `sqlx:"etag"`
}

type AclKeysRow struct {
	ReportId    *string `sqlx:"report_id"`
	SubjectType *string `sqlx:"subject_type"`
	SubjectId   *string `sqlx:"subject_id"`
}
