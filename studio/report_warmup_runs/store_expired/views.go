package store_expired

import (
	time "time"
)

// ExpiredWarmupRun is generated canonical view metadata for warmup_run.
type ExpiredWarmupRun struct {
	RunId     string     `sqlx:"run_id"`
	Status    string     `sqlx:"status"`
	UpdatedAt *time.Time `sqlx:"updated_at"`
}
