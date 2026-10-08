SELECT f.report_id, f.version_no, f.resource_path, f.content
FROM component_resource_files f
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
