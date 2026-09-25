SELECT f.report_id, f.version_no, f.resource_id, f.namespace, f.resource_path,
       f.media_type, f.content, f.content_size, f.content_sha256, f.is_binary
FROM report_resource_files f
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
ORDER BY f.namespace, f.resource_path
