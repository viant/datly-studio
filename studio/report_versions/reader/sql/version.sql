SELECT report_versions."report_id", report_versions."version_no", report_versions."state", report_versions."authoring_mode", report_versions."authored_sql", report_versions."authored_dql", report_versions."component_spec_json", report_versions."spec_format_version", report_versions."spec_hash", report_versions."generated_dql", report_versions."dql_export_limits_json", report_versions."type_manifest_json", report_versions."resource_manifest_json", report_versions."component_descriptor_json", report_versions."compile_status", report_versions."compile_diagnostics_json", report_versions."datly_version", report_versions."compiler_version", report_versions."source_revision", report_versions."notes", report_versions."created_by", report_versions."created_at", report_versions."validated_at", report_versions."published_at" FROM  (
    SELECT v.report_id, v.version_no, v.state, v.authoring_mode,
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
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}

)  report_versions WHERE 1 = 1