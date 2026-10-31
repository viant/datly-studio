SELECT r.report_id, r.resource_check_enabled, r.version_no, r.state, r.compile_status, r.compile_diagnostics_json, r.validated_at, r.source_revision FROM (SELECT version.* FROM  (SELECT v.report_id, v.resource_check_enabled, v.version_no, v.state, v.compile_status,
       v.compile_diagnostics_json, v.validated_at, v.source_revision
FROM component_versions v
)  version) r WHERE $criteria.CompositeIn("r", $VersionKeys)