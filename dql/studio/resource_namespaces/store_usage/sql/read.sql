SELECT u.report_id, u.namespace
FROM (
    SELECT report_id, namespace FROM report_resource_files
    UNION ALL
    SELECT report_id, namespace FROM report_resource_folders
) u
JOIN components workspace_component ON workspace_component.id=u.report_id AND workspace_component.namespace_id=$NamespaceId
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
