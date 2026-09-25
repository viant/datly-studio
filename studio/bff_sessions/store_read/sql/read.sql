SELECT session."session_id_hash", session."subject_id", session."payload_ciphertext", session."expires_at_unix", session."created_at" FROM  (SELECT s.session_id_hash, s.subject_id, s.payload_ciphertext,
       s.expires_at_unix, s.created_at
FROM bff_sessions s
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
)  session WHERE 1 = 1