SELECT r."namespace_id", r."generation_no", r."status", r."report_count", r."activated_at", r."retired_at", r."diagnostics_json" FROM (SELECT generation."namespace_id", generation."generation_no", generation."status", generation."report_count", generation."activated_at", generation."retired_at", generation."diagnostics_json" FROM  (SELECT g.namespace_id, g.generation_no, g.status, g.report_count,
       g.activated_at, g.retired_at, g.diagnostics_json
FROM runtime_generations g
)  generation) r WHERE r.namespace_id=$NamespaceId AND $criteria.CompositeIn("r", $GenerationKeys)