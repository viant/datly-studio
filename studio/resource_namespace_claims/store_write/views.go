package store_write

import (
	time "time"
)

// StoredClaim is generated canonical view metadata for claim.
type StoredClaim struct {
	Namespace    string          `sqlx:"namespace,primaryKey"`
	ReportId     string          `writer:"concurrency" sqlx:"report_id"`
	CreatedAt    time.Time       `sqlx:"created_at"`
	CreatedBy    string          `sqlx:"created_by"`
	UpdatedAt    time.Time       `sqlx:"updated_at"`
	UpdatedBy    string          `sqlx:"updated_by"`
	ShouldDelete bool            `sqlx:"-" writer:"delete"`
	Has          *StoredClaimHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredClaimHas"`
}

type StoredClaimHas struct {
	Namespace    bool
	ReportId     bool
	CreatedAt    bool
	CreatedBy    bool
	UpdatedAt    bool
	UpdatedBy    bool
	ShouldDelete bool
}

// CurrentClaimView is generated canonical view metadata for claim.
type CurrentClaimView struct {
	Namespace string    `sqlx:"namespace,primaryKey"`
	ReportId  string    `sqlx:"report_id"`
	CreatedAt time.Time `sqlx:"created_at"`
	CreatedBy string    `sqlx:"created_by"`
	UpdatedAt time.Time `sqlx:"updated_at"`
	UpdatedBy string    `sqlx:"updated_by"`
}

type ClaimKeysRow struct {
	Namespace string `sqlx:"namespace"`
}
