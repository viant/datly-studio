SELECT COALESCE(MAX(g.generation_no), 0) AS max_generation
FROM runtime_generations g
