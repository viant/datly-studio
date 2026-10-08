SELECT s.report_id, s.version_no, s.skill_id, s.skill_root, f.namespace, f.root_path
FROM component_skill_roots s
JOIN component_resource_folders f
  ON f.report_id = s.report_id AND f.version_no = s.version_no AND f.folder_id = s.folder_id
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
