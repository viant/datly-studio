SELECT grant."report_id", grant."subject_id", grant."subject_type", grant."grant_source", grant."can_view", grant."can_edit", grant."can_publish", grant."is_live" FROM  (SELECT g.report_id, g.subject_id, g.subject_type, g.grant_source,
       g.can_view, g.can_edit, g.can_publish, g.is_live
FROM (
    SELECT r.id AS report_id, r.owner_id AS subject_id, 'user' AS subject_type,
           'owner' AS grant_source, TRUE AS can_view, TRUE AS can_edit,
           TRUE AS can_publish,
           CASE WHEN r.deleted_at IS NULL THEN TRUE ELSE FALSE END AS is_live
    FROM reports r
    UNION ALL
    SELECT a.report_id, a.subject_id, a.subject_type, 'acl' AS grant_source,
           a.can_view, a.can_edit, a.can_publish,
           CASE WHEN r.deleted_at IS NULL THEN TRUE ELSE FALSE END AS is_live
    FROM report_acl a JOIN reports r ON r.id = a.report_id
) g
${predicate.Builder().CombineAnd($predicate.FilterGroup(1, "AND")).Build("WHERE")}
)  grant WHERE 1 = 1