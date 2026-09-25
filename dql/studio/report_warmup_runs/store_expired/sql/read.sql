SELECT w.run_id, w.status, w.updated_at
FROM report_warmup_runs w
WHERE w.status IN ('accepted', 'running') AND w.requested_at < $Before
ORDER BY w.requested_at ASC, w.run_id ASC
LIMIT $PageLimit
