SELECT file."report_id", file."version_no", file."resource_id", file."namespace", file."resource_path", file."media_type", file."content", file."content_size", file."content_sha256", file."is_binary" FROM  (SELECT f.report_id, f.version_no, f.resource_id, f.namespace, f.resource_path,
       f.media_type, f.content, f.content_size, f.content_sha256, f.is_binary
FROM report_resource_files f
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
ORDER BY f.namespace, f.resource_path
)  file WHERE 1 = 1 ORDER BY file.namespace, file.resource_path