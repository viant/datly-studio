SELECT generation."generation_no", generation."source_revision", generation."status", generation."report_count", generation."build_manifest_json", generation."requested_by", generation."requested_at" FROM  (SELECT g.generation_no, g.source_revision, g.status, g.report_count,
       g.build_manifest_json, g.requested_by, g.requested_at
FROM runtime_generations g
)  generation