SELECT file."report_id", file."version_no", file."resource_path", file."content" FROM  (SELECT f.report_id, f.version_no, f.resource_path, f.content
FROM component_resource_files f
WHERE f.namespace <> '__datly_compiled'
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("AND")}
)  file WHERE 1 = 1
