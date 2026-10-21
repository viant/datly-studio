-- Explicit additive upgrade from canonical version 23. Back up before applying.
ALTER TABLE namespaces ADD COLUMN audit_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE components ADD COLUMN audit_enabled BOOLEAN NOT NULL DEFAULT TRUE;
CREATE TABLE tool_call_audit (
    id BIGINT NOT NULL AUTO_INCREMENT,
    namespace_id VARCHAR(64) NULL,
    component_id VARCHAR(64) NULL,
    tool_name VARCHAR(255) NOT NULL,
    started_at DATETIME(6) NOT NULL,
    duration_ms BIGINT NULL,
    status VARCHAR(32) NOT NULL,
    client_ip VARCHAR(45) NULL,
    jwt_sub VARCHAR(255) NULL,
    user_id VARCHAR(255) NULL,
    trace_id VARCHAR(32) NULL,
    span_id VARCHAR(16) NULL,
    request_body LONGBLOB NULL,
    request_compression VARCHAR(8) NOT NULL DEFAULT 'none',
    request_original_bytes BIGINT NULL,
    response_body LONGBLOB NULL,
    response_compression VARCHAR(8) NOT NULL DEFAULT 'none',
    response_original_bytes BIGINT NULL,
    response_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_audit_started ON tool_call_audit(started_at);
CREATE INDEX idx_audit_namespace_tool ON tool_call_audit(namespace_id,tool_name,started_at);
CREATE INDEX idx_audit_subject ON tool_call_audit(jwt_sub,started_at);
CREATE INDEX idx_audit_trace ON tool_call_audit(trace_id,span_id);
CREATE TABLE tool_audit_settings (
    namespace_id VARCHAR(64) NOT NULL,
    tool_name VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL,
    created_by VARCHAR(128) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    updated_by VARCHAR(128) NOT NULL,
    PRIMARY KEY (namespace_id, tool_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

UPDATE schema_version SET version=24;
