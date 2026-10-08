SELECT w.run_id, w.report_id, w.version_no, w.source_revision, w.spec_hash,
       w.plan_key, w.status, w.requested_by, w.requested_at,
       w.created_at, w.created_by, w.updated_at, w.updated_by, w.started_at,
       w.completed_at, w.planned_cases, w.completed_cases, w.max_cases,
       w.row_limit, w.entries, w.duration_ns, w.target_json, w.diagnostics_json
FROM component_warmup_runs w
JOIN components r ON r.id = w.report_id AND r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
