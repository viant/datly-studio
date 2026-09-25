SELECT version."report_id", version."version_no", version."state", version."authoring_mode", version."authored_sql", version."authored_dql", version."component_spec_json", version."spec_format_version", version."spec_hash", version."generated_dql", version."type_manifest_json", version."compile_status", version."datly_version", version."compiler_version", version."source_revision", version."notes", version."created_by", version."created_at" FROM  (SELECT v.report_id, v.version_no, v.state, v.authoring_mode,
       v.authored_sql, v.authored_dql, v.component_spec_json,
       v.spec_format_version, v.spec_hash, v.generated_dql,
       v.type_manifest_json, v.compile_status, v.datly_version,
       v.compiler_version, v.source_revision, v.notes, v.created_by, v.created_at
FROM report_versions v
)  version