SELECT publication."report_id", publication."report_namespace", publication."report_slug", publication."report_title", publication."report_description", publication."report_owner_id", publication."report_status", publication."report_connector", publication."report_component_scope", publication."report_component_name", publication."report_draft_version", publication."report_etag", publication."report_created_at", publication."report_updated_at", publication."publication_active_version_no", publication."publication_desired_generation", publication."publication_active_generation", publication."publication_status", publication."publication_runtime_revision", publication."publication_published_at", publication."version_state", publication."version_authoring_mode", publication."version_authored_sql", publication."version_authored_dql", publication."version_component_spec_json", publication."version_spec_format_version", publication."version_spec_hash", publication."version_generated_dql", publication."version_dql_export_limits_json", publication."version_type_manifest_json", publication."version_resource_manifest_json", publication."version_component_descriptor_json", publication."version_compile_status", publication."version_compile_diagnostics_json", publication."version_datly_version", publication."version_compiler_version", publication."version_source_revision", publication."version_notes", publication."version_created_by", publication."version_created_at", publication."version_validated_at", publication."version_published_at", publication."report_live" FROM  (SELECT c.* FROM (
    SELECT r.id AS report_id, r.namespace AS report_namespace,
           r.slug AS report_slug, r.title AS report_title,
           r.description AS report_description, r.owner_id AS report_owner_id,
           r.status AS report_status, r.default_connector_name AS report_connector,
           r.component_scope AS report_component_scope,
           r.component_name AS report_component_name,
           r.current_draft_version AS report_draft_version,
           r.etag AS report_etag, r.created_at AS report_created_at,
           r.updated_at AS report_updated_at,
           p.active_version_no AS publication_active_version_no,
           p.desired_generation AS publication_desired_generation,
           p.active_generation AS publication_active_generation,
           p.publication_status, p.runtime_revision AS publication_runtime_revision,
           p.published_at AS publication_published_at,
           v.state AS version_state, v.authoring_mode AS version_authoring_mode,
           v.authored_sql AS version_authored_sql,
           v.authored_dql AS version_authored_dql,
           v.component_spec_json AS version_component_spec_json,
           v.spec_format_version AS version_spec_format_version,
           v.spec_hash AS version_spec_hash,
           v.generated_dql AS version_generated_dql,
           v.dql_export_limits_json AS version_dql_export_limits_json,
           v.type_manifest_json AS version_type_manifest_json,
           v.resource_manifest_json AS version_resource_manifest_json,
           v.component_descriptor_json AS version_component_descriptor_json,
           v.compile_status AS version_compile_status,
           v.compile_diagnostics_json AS version_compile_diagnostics_json,
           v.datly_version AS version_datly_version,
           v.compiler_version AS version_compiler_version,
           v.source_revision AS version_source_revision,
           v.notes AS version_notes, v.created_by AS version_created_by,
           v.created_at AS version_created_at,
           v.validated_at AS version_validated_at,
           v.published_at AS version_published_at,
           CASE WHEN r.deleted_at IS NULL THEN TRUE ELSE FALSE END AS report_live
    FROM report_publications p
    JOIN reports r ON r.id = p.report_id
    JOIN report_versions v ON v.report_id = p.report_id AND v.version_no = p.active_version_no
) c
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND")).Build("WHERE")}
)  publication WHERE 1 = 1 ORDER BY publication.publication_published_at DESC, publication.report_id ASC