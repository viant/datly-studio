SELECT r."generation_no", r."status", r."report_count", r."activated_at", r."retired_at", r."diagnostics_json" FROM (SELECT generation."generation_no", generation."status", generation."report_count", generation."activated_at", generation."retired_at", generation."diagnostics_json" FROM  (SELECT g.generation_no, g.status, g.report_count,
       g.activated_at, g.retired_at, g.diagnostics_json
FROM runtime_generations g
)  generation) r WHERE $criteria.CompositeIn("r", $GenerationKeys)