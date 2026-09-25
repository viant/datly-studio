SELECT file."report_id", file."version_no", file."resource_id", file."namespace", file."resource_path", file."content", file."content_size", file."content_sha256", file."is_binary", file."created_at" FROM (SELECT f.report_id, f.version_no, f.resource_id, f.namespace, f.resource_path, f.content,
       f.content_size, f.content_sha256, f.is_binary, f.created_at
FROM report_resource_files f
) file