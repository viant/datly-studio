SELECT f.report_id, f.version_no, f.view_id, f.field_name,
       f.source_column, f.expression, f.database_type, f.go_type,
       f.nullable, f.ordinal, f.filterable, f.orderable,
       f.groupable, f.measurable, f.metadata_json
FROM report_fields f
