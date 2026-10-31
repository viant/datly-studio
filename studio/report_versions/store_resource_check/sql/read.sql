SELECT version.* FROM  (SELECT v.report_id, v.resource_check_enabled, v.version_no, v.state, v.compile_status,
       v.compile_diagnostics_json, v.validated_at, v.source_revision
FROM component_versions v
)  version