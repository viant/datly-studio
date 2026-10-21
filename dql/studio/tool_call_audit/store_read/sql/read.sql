SELECT a.id, a.namespace_id, a.component_id, a.tool_name, a.started_at,
 a.duration_ms, a.status, a.client_ip, a.jwt_sub,
 CASE WHEN $IncludeBodies THEN a.request_body ELSE NULL END AS request_body,
 a.request_compression,
 CASE WHEN $IncludeBodies THEN a.response_body ELSE NULL END AS response_body,
 a.response_compression, a.response_truncated
FROM tool_call_audit a
WHERE COALESCE(a.namespace_id,'') = $NamespaceId
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("AND")}

ORDER BY a.id DESC LIMIT 101
