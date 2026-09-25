SELECT r."report_id", r."version_no", r."state", r."published_at" FROM (SELECT version."report_id", version."version_no", version."state", version."published_at" FROM  (SELECT v.report_id, v.version_no, v.state, v.published_at
FROM report_versions v
)  version) r WHERE $criteria.CompositeIn("r", $VersionKeys)