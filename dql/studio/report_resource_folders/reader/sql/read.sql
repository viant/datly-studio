SELECT f.report_id, f.version_no, f.folder_id, f.namespace, f.root_path, f.uri_prefix, f.ordinal
FROM report_resource_folders f
WHERE f.report_id = $ReportId AND f.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
