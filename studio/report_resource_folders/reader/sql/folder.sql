SELECT resource_folder."report_id", resource_folder."version_no", resource_folder."folder_id", resource_folder."namespace", resource_folder."root_path", resource_folder."uri_prefix", resource_folder."ordinal" FROM  (SELECT f.report_id, f.version_no, f.folder_id, f.namespace, f.root_path, f.uri_prefix, f.ordinal
FROM report_resource_folders f
WHERE f.report_id = $ReportId AND f.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
)  resource_folder WHERE 1 = 1