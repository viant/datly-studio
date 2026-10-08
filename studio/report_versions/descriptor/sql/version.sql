SELECT version."report_id", version."version_no", version."component_spec_json", version."type_manifest_json", version."resource_manifest_json" FROM  (SELECT v.report_id, v.version_no, v.component_spec_json,
       v.type_manifest_json, v.resource_manifest_json
FROM component_versions v
JOIN components r ON r.id = v.report_id AND r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
)  version WHERE 1 = 1