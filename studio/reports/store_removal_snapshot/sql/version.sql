SELECT * FROM (SELECT v.report_id, v.namespace_id, v.version_no, v.source_revision, FALSE AS should_delete FROM component_versions v
) version