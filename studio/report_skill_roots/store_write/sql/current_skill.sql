SELECT r."report_id", r."version_no", r."skill_id", r."folder_id", r."skill_root", r."ordinal" FROM (SELECT skill."report_id", skill."version_no", skill."skill_id", skill."folder_id", skill."skill_root", skill."ordinal", skill."should_delete" FROM  (SELECT s.report_id, s.version_no, s.skill_id, s.folder_id,
       s.skill_root, s.ordinal, FALSE AS should_delete
FROM report_skill_roots s
)  skill) r WHERE $criteria.CompositeIn("r", $SkillKeys)