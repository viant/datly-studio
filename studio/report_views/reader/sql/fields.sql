SELECT fields."report_id", fields."version_no", fields."view_id", fields."field_name", fields."source_column", fields."expression", fields."database_type", fields."go_type", fields."nullable", fields."ordinal", fields."filterable", fields."orderable", fields."groupable", fields."measurable", fields."metadata_json" FROM (
    SELECT f.report_id, f.version_no, f.view_id, f.field_name,
       f.source_column, f.expression, f.database_type, f.go_type,
       f.nullable, f.ordinal, f.filterable, f.orderable,
       f.groupable, f.measurable, f.metadata_json
FROM report_fields f

) fields