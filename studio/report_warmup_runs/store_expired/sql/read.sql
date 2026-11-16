SELECT warmup_run.* FROM  (SELECT w.run_id, w.status, w.updated_at
FROM component_warmup_runs w
WHERE w.status IN ('accepted', 'running')
  AND ((w.deadline_at IS NOT NULL AND w.deadline_at <= $Now)
    OR (w.deadline_at IS NULL AND w.requested_at <= $Before))
ORDER BY w.requested_at ASC, w.run_id ASC
LIMIT $PageLimit
)  warmup_run WHERE 1 = 1