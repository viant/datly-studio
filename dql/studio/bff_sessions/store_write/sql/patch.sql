SELECT s.session_id_hash, s.subject_id, s.payload_ciphertext, s.expires_at_unix, s.created_at, '' AS should_delete
FROM bff_sessions s
