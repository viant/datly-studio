// Copy an immutable published reader into an editable version using Studio's
// versioned SDK operations. Resource identities are version-scoped, so folder
// and skill links can be retained without making the published source mutable.
export async function cloneReaderDraft(api, reportId, versionNo) {
  const [source, resources] = await Promise.all([
    api.getVersion(reportId, versionNo),
    api.getResources(reportId, versionNo),
  ]);
  const authoredDql = source?.authoredDql || source?.generatedDql || '';
  if (!authoredDql) throw new Error('This version has no DQL source available to copy.');
  const created = await api.createVersion(reportId, {
    authoringMode: source.authoringMode || 'dql',
    authoredDql,
    authoredSql: source.authoredSql || '',
    componentSpec: source.componentSpec,
    notes: `Draft from v${versionNo}`,
  });
  let revision = created.sourceRevision;
  let latest = created;
  try {
    for (const file of resources?.files ?? []) {
      const result = await api.upsertResourceFile({
        reportId, versionNo: created.versionNo, expectedSourceRevision: revision,
        resourceId: file.resourceId, namespace: file.namespace,
        resourcePath: file.resourcePath, mediaType: file.mediaType,
        content: file.content, isBinary: file.isBinary,
      });
      latest = result.version;
      revision = latest.sourceRevision;
    }
    for (const folder of resources?.folders ?? []) {
      const result = await api.upsertResourceFolder({
        reportId, versionNo: created.versionNo, expectedSourceRevision: revision,
        folderId: folder.folderId, namespace: folder.namespace,
        rootPath: folder.rootPath, uriPrefix: folder.uriPrefix,
        ordinal: folder.ordinal,
      });
      latest = result.version;
      revision = latest.sourceRevision;
    }
    for (const skill of resources?.skills ?? []) {
      const result = await api.upsertSkillRoot({
        reportId, versionNo: created.versionNo, expectedSourceRevision: revision,
        skillId: skill.skillId, folderId: skill.folderId,
        skillRoot: skill.skillRoot, ordinal: skill.ordinal,
      });
      latest = result.version;
      revision = latest.sourceRevision;
    }
    return latest;
  } catch (cause) {
    cause.partialDraftVersionNo = created.versionNo;
    throw cause;
  }
}
