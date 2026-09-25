SELECT version."report_id", version."version_no", version."state", version."authoring_mode", version."authored_sql", version."authored_dql", version."component_spec_json", version."spec_format_version", version."spec_hash", version."generated_dql", version."dql_export_limits_json", version."type_manifest_json", version."resource_manifest_json", version."component_descriptor_json", version."compile_status", version."compile_diagnostics_json", version."datly_version", version."compiler_version", version."source_revision", version."notes", version."created_by", version."created_at", version."validated_at", version."published_at" FROM  (SELECT v.report_id, v.version_no, v.state, v.authoring_mode,
       v.authored_sql, v.authored_dql, v.component_spec_json,
       v.spec_format_version, v.spec_hash, v.generated_dql,
       v.dql_export_limits_json, v.type_manifest_json,
       v.resource_manifest_json, v.component_descriptor_json,
       v.compile_status, v.compile_diagnostics_json,
       v.datly_version, v.compiler_version, v.source_revision,
       v.notes, v.created_by, v.created_at, v.validated_at, v.published_at
FROM report_versions v
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND")
).Build("WHERE")}
)  version WHERE 1 = 1 ORDER BY version.version_no DESC