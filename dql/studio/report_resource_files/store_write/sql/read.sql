SELECT f.report_id, f.version_no, f.resource_id, f.namespace,
       f.resource_path, f.media_type, f.content, f.content_size,
       f.content_sha256, f.is_binary, f.created_at,
       FALSE AS should_delete
FROM report_resource_files f
