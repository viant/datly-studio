SELECT w.run_id,w.report_id,w.version_no,w.status FROM component_warmup_runs w
WHERE w.report_id = $ReportId
  AND ($TargetVersion = 0 OR w.version_no = $TargetVersion)