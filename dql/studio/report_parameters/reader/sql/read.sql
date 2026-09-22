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
