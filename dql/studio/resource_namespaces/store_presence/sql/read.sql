SELECT u.report_id, u.namespace
FROM (
    SELECT report_id, namespace FROM component_resource_files
    UNION ALL
    SELECT report_id, namespace FROM component_resource_folders
) u
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND")).Build("WHERE")}
