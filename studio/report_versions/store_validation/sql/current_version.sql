SELECT r."report_id", r."version_no", r."compile_status", r."compile_diagnostics_json", r."validated_at", r."source_revision" FROM (SELECT version."report_id", version."version_no", version."compile_status", version."compile_diagnostics_json", version."validated_at", version."source_revision" FROM  (SELECT v.report_id, v.version_no, v.compile_status,
       v.compile_diagnostics_json, v.validated_at, v.source_revision
FROM report_versions v
)  version) r WHERE $criteria.CompositeIn("r", $VersionKeys)