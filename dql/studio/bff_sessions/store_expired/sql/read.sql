SELECT s.session_id_hash
FROM bff_sessions s
WHERE s.expires_at_unix <= $ExpiresBefore
ORDER BY s.expires_at_unix, s.session_id_hash
LIMIT 256
