SELECT g.generation_no, g.source_revision, g.status, g.report_count,
       g.build_manifest_json, g.diagnostics_json, g.requested_by,
       g.requested_at, g.activated_at, g.retired_at
FROM runtime_generations g
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND"), $predicate.FilterGroup(3, "AND")).Build("WHERE")}
