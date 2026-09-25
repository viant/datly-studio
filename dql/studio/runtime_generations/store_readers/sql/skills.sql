SELECT s.report_id, s.version_no, s.skill_id, s.skill_root, f.uri_prefix
FROM report_skill_roots s
JOIN report_resource_folders f ON f.report_id = s.report_id
  AND f.version_no = s.version_no AND f.folder_id = s.folder_id
ORDER BY s.report_id, s.ordinal, s.skill_id
