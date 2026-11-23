-- Apply to layout 29 with writers stopped; existing warmup history is unchanged.
CREATE TABLE component_warmup_schedules (
    schedule_id VARCHAR(64) NOT NULL,
    namespace_id VARCHAR(64) NOT NULL,
    report_id VARCHAR(64) NOT NULL,
    version_no INT NOT NULL,
    interval_seconds BIGINT NOT NULL,
    first_run_at DATETIME NOT NULL,
    next_due_at DATETIME NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    revision BIGINT NOT NULL DEFAULT 1,
    last_run_id VARCHAR(64) NULL,
    created_at DATETIME NOT NULL,
    created_by VARCHAR(128) NOT NULL,
    updated_at DATETIME NOT NULL,
    updated_by VARCHAR(128) NOT NULL,
    PRIMARY KEY (schedule_id),
    UNIQUE (report_id, version_no),
    CONSTRAINT fk_component_warmup_schedules_version FOREIGN KEY (report_id, version_no) REFERENCES component_versions(report_id, version_no),
    CONSTRAINT chk_component_warmup_schedules_interval CHECK (interval_seconds > 0),
    CONSTRAINT chk_component_warmup_schedules_revision CHECK (revision > 0)
);
CREATE INDEX idx_component_warmup_schedules_due ON component_warmup_schedules(enabled, next_due_at, schedule_id);
UPDATE schema_version SET version=30;
