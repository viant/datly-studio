-- Apply explicitly to a version-27 store after backup, with writers stopped.
BEGIN IMMEDIATE;
ALTER TABLE components ADD COLUMN last_version_no INT NOT NULL DEFAULT 0;
UPDATE components SET last_version_no = COALESCE((SELECT MAX(version_no) FROM component_versions WHERE report_id=components.id), 0);
-- Immutable final-owner receipt; component identities remain reserved after removal.
CREATE TABLE component_deletions (
    component_id VARCHAR(64) NOT NULL,
    namespace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    deleted_at DATETIME NOT NULL,
    deleted_by VARCHAR(128) NOT NULL,
    PRIMARY KEY (component_id)
);

ALTER TABLE component_publication_events RENAME TO component_publication_events_v27;
DROP INDEX idx_component_publication_events_owner_time;
DROP INDEX idx_component_publication_events_report_time;
CREATE TABLE component_publication_events (
    namespace_id            VARCHAR(64) NOT NULL DEFAULT '',
    event_id                CHAR(64) NOT NULL,
    report_id               VARCHAR(64) NOT NULL,
    owner_id                VARCHAR(128) NOT NULL,
    operation               VARCHAR(32) NOT NULL,
    version_no              INT NULL,
    generation_no           BIGINT NULL,
    status                  VARCHAR(32) NOT NULL,
    requested_by            VARCHAR(128) NOT NULL,
    reason                  VARCHAR(1000) NULL,
    failure_code            VARCHAR(128) NULL,
    failure_message         VARCHAR(1000) NULL,
    occurred_at             DATETIME NOT NULL,
    PRIMARY KEY (event_id),
    CONSTRAINT fk_component_publication_events_generation
        FOREIGN KEY (generation_no) REFERENCES runtime_generations(generation_no),
    CONSTRAINT chk_component_publication_events_operation
        CHECK (operation IN ('publish', 'rollback', 'unpublish')),
    CONSTRAINT chk_component_publication_events_status
        CHECK (status IN ('succeeded', 'failed'))
);

CREATE INDEX idx_component_publication_events_owner_time
    ON component_publication_events(owner_id, occurred_at DESC);

CREATE INDEX idx_component_publication_events_report_time
    ON component_publication_events(report_id, occurred_at DESC);
INSERT INTO component_publication_events SELECT * FROM component_publication_events_v27;
DROP TABLE component_publication_events_v27;
UPDATE schema_version SET version=28;
COMMIT;
