SELECT v.report_id, v.version_no, v.state, v.compile_status,
       v.compile_diagnostics_json, v.validated_at, v.source_revision
FROM report_versions v
