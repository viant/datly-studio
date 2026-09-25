SELECT folder."report_id", folder."version_no", folder."folder_id", folder."namespace", folder."root_path", folder."uri_prefix", folder."ordinal", folder."should_delete" FROM  (SELECT f.report_id, f.version_no, f.folder_id, f.namespace,
       f.root_path, f.uri_prefix, f.ordinal, FALSE AS should_delete
FROM report_resource_folders f
)  folder