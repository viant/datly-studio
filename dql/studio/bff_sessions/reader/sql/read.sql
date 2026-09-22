SELECT s.session_id_hash, s.subject_id, s.expires_at_unix, s.created_at
FROM bff_sessions s
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("WHERE")}
