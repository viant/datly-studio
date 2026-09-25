SELECT folder."report_id", folder."version_no", folder."folder_id", folder."namespace", folder."root_path", folder."uri_prefix", folder."ordinal" FROM  (SELECT f.report_id, f.version_no, f.folder_id, f.namespace, f.root_path,
       f.uri_prefix, f.ordinal
FROM report_resource_folders f
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
ORDER BY f.ordinal, f.folder_id
)  folder WHERE 1 = 1 ORDER BY folder.ordinal, folder.folder_id