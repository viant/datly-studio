SELECT g.generation_no, g.source_revision, g.status, g.report_count,
       g.build_manifest_json, g.requested_by, g.requested_at
FROM runtime_generations g
