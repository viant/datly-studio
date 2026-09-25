SELECT usage."report_id", usage."namespace" FROM  (SELECT u.report_id, u.namespace
FROM (
    SELECT report_id, namespace FROM report_resource_files
    UNION ALL
    SELECT report_id, namespace FROM report_resource_folders
) u
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
)  usage WHERE 1 = 1