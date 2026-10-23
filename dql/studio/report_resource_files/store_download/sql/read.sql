SELECT f.report_id, f.version_no, f.resource_path, f.content
FROM component_resource_files f
WHERE f.namespace <> '__datly_compiled'
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("AND")}
