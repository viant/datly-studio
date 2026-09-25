SELECT s.session_id_hash, s.subject_id, s.payload_ciphertext,
       s.expires_at_unix, s.created_at
FROM bff_sessions s
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
