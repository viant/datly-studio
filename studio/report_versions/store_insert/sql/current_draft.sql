SELECT r.report_id, r.resource_check_enabled, r.namespace_id, r.version_no, r.state, r.authoring_mode, r.authored_sql, r.authored_dql, r.component_spec_json, r.spec_format_version, r.spec_hash, r.generated_dql, r.type_manifest_json, r.compile_status, r.datly_version, r.compiler_version, r.source_revision, r.notes, r.created_by, r.created_at FROM (SELECT * FROM (SELECT v.report_id, v.resource_check_enabled, v.namespace_id, v.version_no, v.state, v.authoring_mode,
       v.authored_sql, v.authored_dql, v.component_spec_json,
       v.spec_format_version, v.spec_hash, v.generated_dql,
       v.type_manifest_json, v.compile_status, v.datly_version,
       v.compiler_version, v.source_revision, v.notes, v.created_by, v.created_at
FROM component_versions v
) draft) r WHERE $criteria.CompositeIn("r", $Unsafe.ProjectCurrentDraftParentKeys($CurrentVersion))