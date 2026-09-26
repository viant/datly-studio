SELECT v.report_id, v.version_no, v.state, v.authoring_mode,
       v.authored_sql,
       CASE WHEN r.owner_id = $Jwt.Subject OR COALESCE(acl.can_use_dql, FALSE) = TRUE
            THEN v.authored_dql ELSE NULL END AS authored_dql,
       v.component_spec_json, v.dql_export_limits_json, v.type_manifest_json,
       v.resource_manifest_json, v.component_descriptor_json,
       v.spec_format_version, v.spec_hash,
       CASE WHEN r.owner_id = $Jwt.Subject OR COALESCE(acl.can_use_dql, FALSE) = TRUE
            THEN v.generated_dql ELSE NULL END AS generated_dql,
       v.compile_status, v.compile_diagnostics_json, v.datly_version,
       v.compiler_version, v.source_revision, v.notes, v.created_by,
       v.created_at, v.validated_at, v.published_at
FROM report_versions v
JOIN reports r ON r.id = v.report_id AND r.deleted_at IS NULL
LEFT JOIN report_acl acl ON acl.report_id = r.id
  AND acl.subject_type = 'user' AND acl.subject_id = $Jwt.Subject
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
