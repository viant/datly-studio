CREATE TABLE tool_call_audit (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 namespace_id TEXT NULL, component_id TEXT NULL, tool_name TEXT NOT NULL,
 started_at DATETIME NOT NULL, duration_ms INTEGER NULL, status TEXT NOT NULL,
 client_ip TEXT NULL, jwt_sub TEXT NULL, user_id TEXT NULL,
 trace_id TEXT NULL, span_id TEXT NULL,
 request_body BLOB NULL, request_compression TEXT NOT NULL DEFAULT 'none', request_original_bytes INTEGER NULL,
 response_body BLOB NULL, response_compression TEXT NOT NULL DEFAULT 'none', response_original_bytes INTEGER NULL, response_truncated BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_audit_started ON tool_call_audit(started_at);
CREATE INDEX idx_audit_namespace_tool ON tool_call_audit(namespace_id,tool_name,started_at);
CREATE INDEX idx_audit_subject ON tool_call_audit(jwt_sub,started_at);
CREATE INDEX idx_audit_trace ON tool_call_audit(trace_id,span_id);
