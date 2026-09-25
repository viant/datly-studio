SELECT r."session_id_hash", r."subject_id", r."payload_ciphertext", r."expires_at_unix", r."created_at" FROM (SELECT session."session_id_hash", session."subject_id", session."payload_ciphertext", session."expires_at_unix", session."created_at", session."should_delete" FROM  (SELECT s.*, '' AS should_delete
FROM bff_sessions s
)  session WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $SessionKeys)