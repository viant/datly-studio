// The native server clones resources and unchanged policies in one managed
// transaction. A failed clone leaves no partial editable version.
export async function cloneReaderDraft(api, reportId, versionNo) {
  const source = await api.getVersion(reportId, versionNo);
  if (!source?.sourceRevision) throw new Error('Source version revision is unavailable. Reload before cloning.');
  return api.cloneVersion(reportId, versionNo, source.sourceRevision);
}
