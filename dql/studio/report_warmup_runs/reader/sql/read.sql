SELECT w.run_id, w.report_id, w.version_no, w.source_revision, w.spec_hash,
       w.plan_key, w.status, w.requested_by, w.cache_name, w.cache_provider,
       w.connector_name, w.index_column, w.planned_cases, w.completed_cases,
       w.max_cases, w.row_limit, w.entries, w.duration_ns, w.target_json,
       w.diagnostics_json, w.requested_at, w.started_at, w.completed_at
FROM report_warmup_runs w
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(3, "AND")).Build("WHERE")}
