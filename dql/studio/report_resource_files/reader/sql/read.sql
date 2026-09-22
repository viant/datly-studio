SELECT f.report_id, f.version_no, f.resource_id, f.namespace, f.resource_path,
       f.media_type, f.content, f.content_size, f.content_sha256,
       f.is_binary, f.created_at
FROM report_resource_files f
WHERE f.report_id = $ReportId AND f.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
