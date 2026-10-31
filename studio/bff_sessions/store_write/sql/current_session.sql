SELECT r.session_id_hash, r.subject_id, r.payload_ciphertext, r.expires_at_unix, r.created_at FROM (SELECT session.* FROM  (SELECT s.session_id_hash, s.subject_id, s.payload_ciphertext, s.expires_at_unix, s.created_at, '' AS should_delete
FROM bff_sessions s
)  session WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $SessionKeys)