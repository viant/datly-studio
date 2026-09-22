SELECT generation."generation_no", generation."source_revision", generation."status", generation."report_count", generation."build_manifest_json", generation."diagnostics_json", generation."requested_by", generation."requested_at", generation."activated_at", generation."retired_at" FROM  (SELECT g.generation_no, g.source_revision, g.status, g.report_count,
       g.build_manifest_json, g.diagnostics_json, g.requested_by,
       g.requested_at, g.activated_at, g.retired_at
FROM runtime_generations g
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(3, "AND")).Build("WHERE")}
)  generation WHERE 1 = 1