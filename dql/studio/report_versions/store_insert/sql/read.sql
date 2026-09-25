SELECT v.report_id, v.version_no, v.state, v.authoring_mode,
       v.authored_sql, v.authored_dql, v.component_spec_json,
       v.spec_format_version, v.spec_hash, v.generated_dql,
       v.type_manifest_json, v.compile_status, v.datly_version,
       v.compiler_version, v.source_revision, v.notes, v.created_by, v.created_at
FROM report_versions v
