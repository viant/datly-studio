SELECT p.*
FROM authorization_predicates p
WHERE p.deleted_at IS NULL
