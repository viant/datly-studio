// Resolve an explicit authored embed reference, never infer a filename from a
// display name. This does not parse SQL or construct the component graph.
export function viewSQLResource(inspection, name, files = []) {
  const root = inspection?.structure?.component?.rootView;
  const find = view => !view ? null : view.name === name || view.namespace === name ? view : (view.relations ?? []).map(relation => find(relation.view)).find(Boolean);
  const view = find(root);
  const authored = (inspection?.structure?.views ?? []).find(item => item.name === name || item.name === view?.namespace)?.sql;
  const sql = authored || view?.source?.sql || '';
  const refs = [...sql.matchAll(/\$\{embed:([^}]+)\}/g)];
  if (refs.length !== 1) return { sql, file: null };
  const path = refs[0][1];
  const candidates = files.filter(file => file.resourcePath === path);
  if (candidates.length !== 1) return { sql: '', file: null, path, error: candidates.length ? 'The SQL resource reference is ambiguous.' : `SQL resource ${path} is unavailable.` };
  return { sql: candidates[0].content, file: candidates[0], path };
}
