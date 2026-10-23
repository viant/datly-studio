-- Explicit additive upgrade from canonical version 24. Back up before applying.
CREATE TABLE artifact_release_bundles (
 scope_id CHAR(64) NOT NULL, tenant_id VARCHAR(512) NOT NULL,
 account_id VARCHAR(128) NOT NULL, workspace_id VARCHAR(128) NOT NULL,
 namespace VARCHAR(200) NOT NULL, bundle_hash CHAR(64) NOT NULL,
 bundle_json JSON NOT NULL, created_at DATETIME(6) NOT NULL,
 created_by VARCHAR(128) NOT NULL, updated_at DATETIME(6) NOT NULL,
 updated_by VARCHAR(128) NOT NULL,
 PRIMARY KEY (scope_id,bundle_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE artifact_release_channels (
 scope_id CHAR(64) NOT NULL, tenant_id VARCHAR(512) NOT NULL,
 account_id VARCHAR(128) NOT NULL, workspace_id VARCHAR(128) NOT NULL,
 namespace VARCHAR(200) NOT NULL, name VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
 description TEXT NOT NULL, policy_refs_json JSON NOT NULL,
 active_bundle_hash CHAR(64) NULL, revision BIGINT NOT NULL,
 pending_operation_id CHAR(32) NULL, created_at DATETIME(6) NOT NULL,
 created_by VARCHAR(128) NOT NULL, updated_at DATETIME(6) NOT NULL,
 updated_by VARCHAR(128) NOT NULL,
 PRIMARY KEY (scope_id,name),
 CONSTRAINT fk_artifact_release_channel_bundle
  FOREIGN KEY (scope_id,active_bundle_hash) REFERENCES artifact_release_bundles(scope_id,bundle_hash),
 CONSTRAINT chk_artifact_release_channel_revision CHECK (revision >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE artifact_release_operations (
 operation_id CHAR(32) NOT NULL, scope_id CHAR(64) NOT NULL,
 channel_name VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL, expected_revision BIGINT NOT NULL,
 target_revision BIGINT NOT NULL, target_bundle_hash CHAR(64) NOT NULL,
 next_channel_json JSON NOT NULL, status VARCHAR(16) NOT NULL,
 created_at DATETIME(6) NOT NULL, created_by VARCHAR(128) NOT NULL,
 updated_at DATETIME(6) NOT NULL, updated_by VARCHAR(128) NOT NULL,
 completed_at DATETIME(6) NULL,
 failure_code VARCHAR(64) NULL,
 PRIMARY KEY (operation_id),
 CONSTRAINT fk_artifact_release_operation_channel
  FOREIGN KEY (scope_id,channel_name) REFERENCES artifact_release_channels(scope_id,name),
 CONSTRAINT fk_artifact_release_operation_bundle
  FOREIGN KEY (scope_id,target_bundle_hash) REFERENCES artifact_release_bundles(scope_id,bundle_hash),
 CONSTRAINT chk_artifact_release_operation_status CHECK (status IN ('pending','applied','failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_artifact_release_operations_pending
 ON artifact_release_operations(scope_id,status,created_at);
UPDATE schema_version SET version=25;
