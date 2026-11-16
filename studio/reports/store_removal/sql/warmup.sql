SELECT * FROM (SELECT w.run_id, w.report_id, w.version_no, w.status, FALSE AS should_delete FROM component_warmup_runs w
) warmup