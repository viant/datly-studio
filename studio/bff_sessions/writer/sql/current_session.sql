SELECT r."session_id_hash", r."subject_id", r."expires_at_unix" FROM (SELECT session."session_id_hash", session."subject_id", session."expires_at_unix" FROM  (SELECT s.session_id_hash, s.subject_id, s.expires_at_unix
FROM bff_sessions s
)  session WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $SessionKeys)