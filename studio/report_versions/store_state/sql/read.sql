SELECT version."report_id", version."version_no", version."state", version."published_at" FROM  (SELECT v.report_id, v.version_no, v.state, v.published_at
FROM report_versions v
)  version