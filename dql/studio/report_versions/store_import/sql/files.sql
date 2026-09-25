SELECT f.report_id, f.version_no, f.resource_id, f.namespace, f.resource_path, f.content,
       f.content_size, f.content_sha256, f.is_binary, f.created_at
FROM report_resource_files f
