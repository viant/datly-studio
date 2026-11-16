SELECT r.namespace_id, r.run_id, r.report_id, r.version_no, r.source_revision, r.spec_hash, r.plan_key, r.active_key, r.status, r.requested_by, r.cache_name, r.cache_provider, r.connector_name, r.index_column, r.planned_cases, r.completed_cases, r.max_cases, r.row_limit, r.entries, r.duration_ns, r.diagnostics_json, r.target_json, r.requested_at, r.deadline_at, r.created_at, r.created_by, r.updated_at, r.updated_by, r.started_at, r.completed_at FROM (SELECT warmup_run.* FROM  (SELECT w.namespace_id, w.run_id, w.report_id, w.version_no, w.source_revision, w.spec_hash,
       w.plan_key, w.active_key, w.status, w.requested_by,
       w.cache_name, w.cache_provider, w.connector_name, w.index_column,
       w.planned_cases, w.completed_cases, w.max_cases, w.row_limit,
       w.entries, w.duration_ns, w.diagnostics_json, w.target_json,
       w.requested_at, w.deadline_at, w.created_at, w.created_by, w.updated_at, w.updated_by,
       w.started_at, w.completed_at
FROM component_warmup_runs w
)  warmup_run WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $WarmupRunKeys)