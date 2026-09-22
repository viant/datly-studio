SELECT parameter."report_id", parameter."version_no", parameter."parameter_id", parameter."parameter_identity", parameter."name", parameter."source_kind", parameter."source_name", parameter."type_expr", parameter."required", parameter."emit_output", parameter."query_selector_json", parameter."codec_json", parameter."activation_json", parameter."metadata_json", parameter."ordinal", parameter."should_delete" FROM  (
    SELECT p.*, '' AS should_delete
FROM report_parameters p

)  parameter WHERE 1 = 1