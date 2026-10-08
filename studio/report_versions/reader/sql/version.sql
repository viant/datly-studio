SELECT component_versions."report_id", component_versions."version_no", component_versions."state", component_versions."authoring_mode", component_versions."authored_sql", component_versions."authored_dql", component_versions."component_spec_json", component_versions."spec_format_version", component_versions."spec_hash", component_versions."generated_dql", component_versions."dql_export_limits_json", component_versions."type_manifest_json", component_versions."resource_manifest_json", component_versions."component_descriptor_json", component_versions."compile_status", component_versions."compile_diagnostics_json", component_versions."datly_version", component_versions."compiler_version", component_versions."source_revision", component_versions."notes", component_versions."created_by", component_versions."created_at", component_versions."validated_at", component_versions."published_at" FROM  (
    SELECT v.report_id, v.version_no, v.state, v.authoring_mode,
       v.authored_sql, v.authored_dql, v.component_spec_json,
       v.spec_format_version, v.spec_hash, v.generated_dql,
       v.dql_export_limits_json, v.type_manifest_json,
       v.resource_manifest_json, v.component_descriptor_json,
       v.compile_status, v.compile_diagnostics_json,
       v.datly_version, v.compiler_version, v.source_revision,
       v.notes, v.created_by, v.created_at, v.validated_at, v.published_at
FROM component_versions v
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}

)  component_versions WHERE 1 = 1