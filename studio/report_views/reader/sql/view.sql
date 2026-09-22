SELECT views."report_id", views."version_no", views."view_id", views."view_identity", views."parent_view_id", views."relation_name", views."name", views."namespace", views."role", views."cardinality", views."connector_name", views."source_kind", views."source_sql", views."source_table", views."metadata_json" FROM  (
    SELECT v.report_id, v.version_no, v.view_id, v.view_identity,
       v.parent_view_id, v.relation_name, v.name, v.namespace,
       v.role, v.cardinality, v.connector_name, v.source_kind,
       v.source_sql, v.source_table, v.metadata_json
FROM report_views v
WHERE v.report_id = $ReportId
  AND v.version_no = $VersionNo
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}

)  views WHERE 1 = 1