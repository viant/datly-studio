SELECT calls.* FROM  (SELECT a.id, a.namespace_id, a.component_id, a.tool_name, a.started_at,
 a.duration_ms, a.status, a.client_ip, a.jwt_sub, a.user_id, a.trace_id, a.span_id,
 a.request_body, a.request_compression, a.request_original_bytes,
 a.response_body, a.response_compression, a.response_truncated
FROM tool_call_audit a
)  calls