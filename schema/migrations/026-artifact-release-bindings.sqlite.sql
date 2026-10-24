-- Explicit additive upgrade from canonical version 25. Back up first.
BEGIN IMMEDIATE;
ALTER TABLE artifact_release_channels ADD COLUMN active_dependencies_json JSON NULL;
ALTER TABLE artifact_release_operations ADD COLUMN dependencies_json JSON NULL;
CREATE TABLE artifact_release_bundle_bindings (
 scope_id CHAR(64) NOT NULL, bundle_hash CHAR(64) NOT NULL,
 bindings_json JSON NOT NULL, bindings_hash CHAR(64) NOT NULL,
 created_at DATETIME NOT NULL, created_by VARCHAR(128) NOT NULL,
 updated_at DATETIME NOT NULL, updated_by VARCHAR(128) NOT NULL,
 PRIMARY KEY (scope_id,bundle_hash),
 FOREIGN KEY (scope_id,bundle_hash) REFERENCES artifact_release_bundles(scope_id,bundle_hash)
);
UPDATE schema_version SET version=26;
COMMIT;
