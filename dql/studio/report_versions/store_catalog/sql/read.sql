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
    $predicate.FilterGroup(2, "AND")
).Build("WHERE")}
