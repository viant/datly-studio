SELECT resource_file."report_id", resource_file."version_no", resource_file."resource_id", resource_file."namespace", resource_file."resource_path", resource_file."media_type", resource_file."content", resource_file."content_size", resource_file."content_sha256", resource_file."is_binary", resource_file."created_at" FROM  (SELECT f.report_id, f.version_no, f.resource_id, f.namespace, f.resource_path,
       f.media_type, f.content, f.content_size, f.content_sha256,
       f.is_binary, f.created_at
FROM report_resource_files f
WHERE f.report_id = $ReportId AND f.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(2, "AND"), $predicate.FilterGroup(3, "AND")).Build("AND")}
)  resource_file WHERE 1 = 1