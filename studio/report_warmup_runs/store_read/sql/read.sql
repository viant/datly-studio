SELECT warmup_run."run_id", warmup_run."report_id", warmup_run."version_no", warmup_run."source_revision", warmup_run."spec_hash", warmup_run."plan_key", warmup_run."status", warmup_run."requested_by", warmup_run."requested_at", warmup_run."created_at", warmup_run."created_by", warmup_run."updated_at", warmup_run."updated_by", warmup_run."started_at", warmup_run."completed_at", warmup_run."planned_cases", warmup_run."completed_cases", warmup_run."max_cases", warmup_run."row_limit", warmup_run."entries", warmup_run."duration_ns", warmup_run."target_json", warmup_run."diagnostics_json" FROM  (SELECT w.run_id, w.report_id, w.version_no, w.source_revision, w.spec_hash,
       w.plan_key, w.status, w.requested_by, w.requested_at,
       w.created_at, w.created_by, w.updated_at, w.updated_by, w.started_at,
       w.completed_at, w.planned_cases, w.completed_cases, w.max_cases,
       w.row_limit, w.entries, w.duration_ns, w.target_json, w.diagnostics_json
FROM report_warmup_runs w
WHERE ($ReportId = '' OR w.report_id = $ReportId)
  AND ($RunId = '' OR w.run_id = $RunId)
  AND ($ActiveKey = '' OR w.active_key = $ActiveKey)
  AND ($VersionNo = 0 OR w.version_no = $VersionNo)
ORDER BY w.requested_at DESC, w.run_id DESC
LIMIT $PageLimit OFFSET $PageOffset
)  warmup_run WHERE 1 = 1