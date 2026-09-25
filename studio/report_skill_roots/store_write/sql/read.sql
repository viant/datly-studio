SELECT skill."report_id", skill."version_no", skill."skill_id", skill."folder_id", skill."skill_root", skill."ordinal", skill."should_delete" FROM  (SELECT s.report_id, s.version_no, s.skill_id, s.folder_id,
       s.skill_root, s.ordinal, FALSE AS should_delete
FROM report_skill_roots s
)  skill