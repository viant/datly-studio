SELECT skill.* FROM  (SELECT s.namespace_id,s.skill_id,s.name,s.published_revision AS revision,r.bundle_json,r.content_sha256
FROM namespace_skills s JOIN namespace_skill_revisions r
 ON r.namespace_id=s.namespace_id AND r.skill_id=s.skill_id AND r.revision=s.published_revision
WHERE s.namespace_id=$NamespaceId AND s.status='active' AND s.deleted_at IS NULL AND s.published_revision>0
ORDER BY s.name
)  skill