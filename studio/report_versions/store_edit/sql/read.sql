SELECT version."report_id", version."version_no", version."state", version."authored_sql", version."authored_dql", version."component_spec_json", version."spec_hash", version."generated_dql", version."compile_status", version."compile_diagnostics_json", version."source_revision" FROM  (SELECT v.report_id, v.version_no, v.state, v.authored_sql, v.authored_dql,
       v.component_spec_json, v.spec_hash, v.generated_dql,
       v.compile_status, v.compile_diagnostics_json, v.source_revision
FROM report_versions v
)  version