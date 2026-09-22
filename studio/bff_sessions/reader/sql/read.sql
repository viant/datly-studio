SELECT session."session_id_hash", session."subject_id", session."expires_at_unix", session."created_at" FROM  (SELECT s.session_id_hash, s.subject_id, s.expires_at_unix, s.created_at
FROM bff_sessions s
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("WHERE")}
)  session WHERE 1 = 1