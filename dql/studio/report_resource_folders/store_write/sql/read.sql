SELECT f.report_id, f.version_no, f.folder_id, f.namespace,
       f.root_path, f.uri_prefix, f.ordinal, FALSE AS should_delete
FROM report_resource_folders f
