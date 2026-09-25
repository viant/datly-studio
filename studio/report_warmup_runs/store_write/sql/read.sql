SELECT warmup_run."run_id", warmup_run."report_id", warmup_run."version_no", warmup_run."source_revision", warmup_run."spec_hash", warmup_run."plan_key", warmup_run."active_key", warmup_run."status", warmup_run."requested_by", warmup_run."cache_name", warmup_run."cache_provider", warmup_run."connector_name", warmup_run."index_column", warmup_run."planned_cases", warmup_run."completed_cases", warmup_run."max_cases", warmup_run."row_limit", warmup_run."entries", warmup_run."duration_ns", warmup_run."diagnostics_json", warmup_run."target_json", warmup_run."requested_at", warmup_run."created_at", warmup_run."created_by", warmup_run."updated_at", warmup_run."updated_by", warmup_run."started_at", warmup_run."completed_at" FROM  (SELECT w.run_id, w.report_id, w.version_no, w.source_revision, w.spec_hash,
       w.plan_key, w.active_key, w.status, w.requested_by,
       w.cache_name, w.cache_provider, w.connector_name, w.index_column,
       w.planned_cases, w.completed_cases, w.max_cases, w.row_limit,
       w.entries, w.duration_ns, w.diagnostics_json, w.target_json,
       w.requested_at, w.created_at, w.created_by, w.updated_at, w.updated_by,
       w.started_at, w.completed_at
FROM report_warmup_runs w
)  warmup_run WHERE 1 = 1