SELECT n.*, '' AS should_delete
FROM namespaces n
WHERE n.owner_id = $Auth.Auth.Subject
