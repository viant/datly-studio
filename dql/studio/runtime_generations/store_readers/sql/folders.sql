SELECT f.report_id, f.version_no, f.folder_id, f.namespace,
       f.root_path, f.uri_prefix
FROM report_resource_folders f
ORDER BY f.report_id, f.ordinal, f.folder_id
