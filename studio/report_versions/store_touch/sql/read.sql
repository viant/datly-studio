SELECT version."report_id", version."version_no", version."state", version."compile_status", version."compile_diagnostics_json", version."validated_at", version."source_revision" FROM  (SELECT v.report_id, v.version_no, v.state, v.compile_status,
       v.compile_diagnostics_json, v.validated_at, v.source_revision
FROM report_versions v
)  version