SELECT g.generation_no, g.status, g.report_count,
       g.activated_at, g.retired_at, g.diagnostics_json
FROM runtime_generations g
