-- Apply explicitly to a version-27 store after backup, with writers stopped.
-- MySQL DDL commits implicitly; complete all statements before restarting Studio.
ALTER TABLE components ADD COLUMN last_version_no INT NOT NULL DEFAULT 0;
UPDATE components SET last_version_no = COALESCE((SELECT MAX(version_no) FROM component_versions WHERE report_id=components.id), 0);
-- Immutable final-owner receipt; component identities remain reserved after removal.
CREATE TABLE component_deletions (
    component_id VARCHAR(64) NOT NULL,
    namespace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    deleted_at DATETIME(6) NOT NULL,
    deleted_by VARCHAR(128) NOT NULL,
    PRIMARY KEY (component_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE component_publication_events
    DROP FOREIGN KEY fk_component_publication_events_report,
    DROP FOREIGN KEY fk_component_publication_events_version;
UPDATE schema_version SET version=28;
