SELECT file."report_id", file."version_no", file."resource_id", file."namespace", file."resource_path", file."media_type", file."content", file."content_size", file."content_sha256", file."is_binary", file."created_at", file."should_delete" FROM  (SELECT f.report_id, f.version_no, f.resource_id, f.namespace,
       f.resource_path, f.media_type, f.content, f.content_size,
       f.content_sha256, f.is_binary, f.created_at,
       FALSE AS should_delete
FROM report_resource_files f
)  file