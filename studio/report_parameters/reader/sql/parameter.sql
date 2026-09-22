SELECT parameters."report_id", parameters."version_no", parameters."parameter_id", parameters."parameter_identity", parameters."name", parameters."source_kind", parameters."source_name", parameters."type_expr", parameters."required", parameters."emit_output", parameters."query_selector_json", parameters."codec_json", parameters."activation_json", parameters."metadata_json" FROM  (
    SELECT p.report_id, p.version_no, p.parameter_id, p.parameter_identity,
       p.name, p.source_kind, p.source_name, p.type_expr, p.required,
       p.emit_output, p.query_selector_json, p.codec_json,
       p.activation_json, p.metadata_json
FROM report_parameters p
WHERE p.report_id = $ReportId
  AND p.version_no = $VersionNo
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}

)  parameters WHERE 1 = 1