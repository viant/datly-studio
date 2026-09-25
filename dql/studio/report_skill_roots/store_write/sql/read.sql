SELECT s.report_id, s.version_no, s.skill_id, s.folder_id,
       s.skill_root, s.ordinal, FALSE AS should_delete
FROM report_skill_roots s
