SELECT r.report_id, r.namespace_id, r.version_no, r.resource_id, r.namespace, r.resource_path, r.media_type, r.content, r.content_size, r.content_sha256, r.is_binary, r.created_at FROM (SELECT file.* FROM  (SELECT f.report_id, f.namespace_id, f.version_no, f.resource_id, f.namespace,
       f.resource_path, f.media_type, f.content, f.content_size,
       f.content_sha256, f.is_binary, f.created_at,
       FALSE AS should_delete
FROM component_resource_files f
)  file) r WHERE $criteria.CompositeIn("r", $FileKeys)