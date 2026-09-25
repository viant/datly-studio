SELECT w.run_id, w.report_id, w.version_no, w.source_revision, w.spec_hash,
       w.plan_key, w.active_key, w.status, w.requested_by,
       w.cache_name, w.cache_provider, w.connector_name, w.index_column,
       w.planned_cases, w.completed_cases, w.max_cases, w.row_limit,
       w.entries, w.duration_ns, w.diagnostics_json, w.target_json,
       w.requested_at, w.created_at, w.created_by, w.updated_at, w.updated_by,
       w.started_at, w.completed_at
FROM report_warmup_runs w
