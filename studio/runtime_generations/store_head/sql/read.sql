SELECT head."max_generation" FROM  (SELECT COALESCE(MAX(g.generation_no), 0) AS max_generation
FROM runtime_generations g
)  head WHERE 1 = 1