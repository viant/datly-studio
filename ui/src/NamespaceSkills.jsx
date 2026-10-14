import React, { useEffect, useRef, useState } from 'react';
import { Button, Callout, FormGroup, HTMLTable, InputGroup, Spinner, TextArea } from '@blueprintjs/core';

const PAGE_SIZE = 20;
const initialBundle = () => ({ title: '', content: '---\nname: namespace-guide\ndescription: Describe when to use this namespace skill.\n---\n\nWrite the instructions here.\n', files: [] });
const copyBundle = bundle => ({ title: bundle?.title || '', content: bundle?.content || '', files: (bundle?.files || []).map(file => ({ ...file })) });

export function NamespaceSkills({ api, namespace, onDirtyChange }) {
  const [rows, setRows] = useState([]), [loading, setLoading] = useState(true);
  const [query, setQuery] = useState(''), [search, setSearch] = useState(''), [offset, setOffset] = useState(0);
  const [hasNext, setHasNext] = useState(false), [refresh, setRefresh] = useState(0);
  const [error, setError] = useState(''), [notice, setNotice] = useState(''), [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState(null), [bundle, setBundle] = useState(initialBundle);
  const [dirty, setDirty] = useState(false), [conflict, setConflict] = useState(false);
  const [history, setHistory] = useState(null), [preview, setPreview] = useState(null);
  const alive = useRef(true);
  const scopeID = namespace?.namespaceId;
  const activeScope = useRef(scopeID);
  activeScope.current = scopeID;
  const isCurrent = () => alive.current && activeScope.current === scopeID;
  useEffect(() => {
    setEditing(null); setDirty(false); setBusy(false); setHistory(null); setPreview(null);
    setError(''); setNotice(''); setConflict(false); setOffset(0); setQuery(''); setSearch('');
  }, [scopeID]);
  const editable = namespace?.canManage !== false;
  const publishingEnabled = Boolean(namespace?.mcpEnabled);
  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  useEffect(() => { onDirtyChange?.(dirty, { busy }); }, [dirty, busy, onDirtyChange]);
  useEffect(() => () => onDirtyChange?.(false), [onDirtyChange]);
  useEffect(() => {
    let cancelled = false;
    setLoading(true); setError('');
    if (!editable) { setRows([]); setLoading(false); return; }
    api.listNamespaceSkills({ query: search, limit: PAGE_SIZE + 1, offset })
      .then(page => { if (!cancelled) { const items = page?.items || []; setRows(items.slice(0, PAGE_SIZE)); setHasNext(items.length > PAGE_SIZE); } })
      .catch(cause => { if (!cancelled && cause.name !== 'AbortError') setError(cause.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [api, namespace?.namespaceId, editable, search, offset, refresh]);

  const run = async operation => {
    if (busy) return;
    setBusy(true); setError(''); setNotice(''); setConflict(false);
    try { await operation(); }
    catch (cause) { if (isCurrent() && cause.name !== 'AbortError') { setError(cause.message); setConflict(cause.code === 'conflict'); } }
    finally { if (isCurrent()) setBusy(false); }
  };
  const edit = skill => run(async () => {
    const result = await api.getNamespaceSkill(skill.skillId);
    if (!isCurrent()) return;
    setEditing(result.skill); setBundle(copyBundle(result.revision.bundle)); setDirty(false); setHistory(null); setPreview(null);
  });
  const create = () => { setEditing({ new: true }); setBundle(initialBundle()); setDirty(true); setError(''); setConflict(false); setNotice(''); setHistory(null); setPreview(null); };
  const change = value => { setBundle(value); setDirty(true); };
  const discard = () => { setEditing(null); setDirty(false); setError(''); setConflict(false); };
  const save = event => {
    event.preventDefault();
    if (!editable || !bundle.content.trim()) return;
    run(async () => {
      const saved = editing.new ? await api.createNamespaceSkill(bundle) : await api.updateNamespaceSkill(editing.skillId, editing.etag, bundle);
      if (!isCurrent()) return;
      setEditing(null); setDirty(false); setNotice(`Saved ${saved.skill.name}, revision ${saved.skill.latestRevision}.`); setRefresh(value => value + 1);
    });
  };
  const publish = skill => run(async () => {
    await api.publishNamespaceSkill(skill.skillId, skill.etag, skill.latestRevision);
    if (isCurrent()) { setNotice(`Published ${skill.name}, revision ${skill.latestRevision}.`); setRefresh(value => value + 1); }
  });
  const unpublish = skill => run(async () => {
    await api.unpublishNamespaceSkill(skill.skillId, skill.etag);
    if (isCurrent()) { setNotice(`Unpublished ${skill.name}.`); setRefresh(value => value + 1); }
  });
  const archive = skill => run(async () => {
    await api.deleteNamespaceSkill(skill.skillId, skill.etag);
    if (isCurrent()) { setNotice(`Archived ${skill.name}.`); setOffset(0); setRefresh(value => value + 1); }
  });
  const revisions = skill => run(async () => {
    const result = await api.listNamespaceSkillRevisions(skill.skillId, { limit: 100 });
    if (isCurrent()) { setHistory({ skill, items: result.items || [] }); setPreview(null); }
  });

  return <section aria-label="Namespace-owned skills" className="studio-namespace-skills">
    <p>These skills belong to the namespace. Save drafts independently of components; publish a saved revision to make it available through MCP.</p>
    {!editable && <Callout>Namespace skills are managed by the namespace owner and namespace administrators.</Callout>}
    {notice && <p role="status">{notice}</p>}
    {error && <Callout intent={conflict ? 'warning' : 'danger'} role="alert" title={conflict ? 'Skill changed elsewhere' : 'Namespace skill action failed'}>
      <p>{error}</p>
      {conflict && editing && !editing.new && <><p>Your edits are preserved. Reloading discards them and opens the latest saved revision.</p><Button disabled={busy} onClick={() => edit(editing)}>Discard edits and reload</Button></>}
      {!editing && <Button disabled={busy} onClick={() => setRefresh(value => value + 1)}>Refresh skills</Button>}
    </Callout>}
    {editing ? <form onSubmit={save} className="studio-namespace-skill-editor">
      <h2>{editing.new ? 'New namespace skill' : `Edit ${editing.name}`}</h2>
      {!editing.new && <p>Editing revision {editing.latestRevision}. Saving creates a new revision; publication stays unchanged until you publish it.</p>}
      <FormGroup label="Display title" labelFor="namespace-skill-title"><InputGroup id="namespace-skill-title" value={bundle.title} onChange={event => change({ ...bundle, title: event.target.value })} disabled={busy} maxLength={300}/></FormGroup>
      <FormGroup label="SKILL.md" labelFor="namespace-skill-content" helperText="Include YAML frontmatter with name and description, followed by Markdown instructions. The name cannot change after creation.">
        <TextArea id="namespace-skill-content" className="studio-code-input" fill rows={15} value={bundle.content} disabled={busy} onChange={event => change({ ...bundle, content: event.target.value })}/>
      </FormGroup>
      <h3>Supporting files</h3>
      {bundle.files.map((file, index) => <fieldset key={index} className="studio-skill-file"><legend>File {index + 1}</legend>
        <FormGroup label="Relative path" labelFor={`namespace-skill-file-path-${index}`}><InputGroup id={`namespace-skill-file-path-${index}`} aria-label={`File path ${index + 1}`} value={file.path} disabled={busy || file.isBinary} onChange={event => change({ ...bundle, files: bundle.files.map((item, position) => position === index ? { ...item, path: event.target.value } : item) })}/></FormGroup>
        {file.isBinary ? <p>Existing binary content will be preserved.</p> : <FormGroup label="Content" labelFor={`namespace-skill-file-content-${index}`}><TextArea id={`namespace-skill-file-content-${index}`} aria-label={`File content ${index + 1}`} fill rows={4} value={file.content} disabled={busy} onChange={event => change({ ...bundle, files: bundle.files.map((item, position) => position === index ? { ...item, content: event.target.value } : item) })}/></FormGroup>}
        <Button icon="remove" disabled={busy} aria-label={`Remove supporting file ${index + 1}`} onClick={() => change({ ...bundle, files: bundle.files.filter((_, position) => position !== index) })}>Remove file</Button>
      </fieldset>)}
      <Button icon="add" disabled={busy} onClick={() => change({ ...bundle, files: [...bundle.files, { path: '', content: '' }] })}>Add supporting file</Button>
      <div className="studio-skill-actions"><Button type="submit" intent="primary" loading={busy} disabled={!dirty || !bundle.content.trim() || conflict}>Save namespace skill</Button><Button disabled={busy} onClick={discard}>{dirty ? 'Discard changes' : 'Back to skills'}</Button></div>
    </form> : <>
      {editable && <div className="studio-skill-toolbar"><form className="studio-skill-search" onSubmit={event => { event.preventDefault(); setSearch(query); setOffset(0); }}><InputGroup aria-label="Search namespace skills" placeholder="Find a skill" value={query} disabled={busy} onChange={event => setQuery(event.target.value)}/><Button type="submit" disabled={busy}>Search</Button></form><Button icon="add" intent="primary" disabled={busy} onClick={create}>New namespace skill</Button><Button icon="refresh" disabled={busy || loading} onClick={() => setRefresh(value => value + 1)}>Refresh</Button></div>}
      {editable && !publishingEnabled && <p>Enable MCP in namespace settings when you are ready to publish. Draft editing is available now.</p>}
      {loading ? <Spinner size={22} aria-label="Loading namespace skills"/> : rows.length ? <div className="studio-table-wrap"><HTMLTable className="studio-data-table" striped><thead><tr><th>Skill</th><th>Draft revision</th><th>Publication</th><th>Actions</th></tr></thead><tbody>{rows.map(skill => <tr key={skill.skillId}>
        <td><strong>{skill.title || skill.name}</strong><p>{skill.description}</p></td><td>{skill.latestRevision}</td><td>{skill.publishedRevision ? `Published revision ${skill.publishedRevision}` : 'Draft only'}</td>
        <td><div className="studio-skill-row-actions"><Button small icon="edit" disabled={busy} aria-label={`Edit ${skill.name}`} onClick={() => edit(skill)}>Edit</Button><Button small disabled={busy} aria-label={`History of ${skill.name}`} onClick={() => revisions(skill)}>History</Button>
          <Button small disabled={busy || !publishingEnabled || skill.publishedRevision === skill.latestRevision} aria-label={`Publish ${skill.name}`} onClick={() => publish(skill)}>Publish</Button>
          {skill.publishedRevision ? <Button small disabled={busy} aria-label={`Unpublish ${skill.name}`} onClick={() => unpublish(skill)}>Unpublish</Button> : <Button small disabled={busy} aria-label={`Archive ${skill.name}`} onClick={() => archive(skill)}>Archive</Button>}
        </div></td>
      </tr>)}</tbody></HTMLTable></div> : editable && !error && <p>No namespace skills found. Create a skill to add reusable instructions.</p>}
      {editable && <div className="studio-skill-pagination"><Button disabled={busy || loading || offset === 0} onClick={() => setOffset(value => Math.max(0, value - PAGE_SIZE))}>Previous</Button><span>Page {Math.floor(offset / PAGE_SIZE) + 1}</span><Button disabled={busy || loading || !hasNext} onClick={() => setOffset(value => value + PAGE_SIZE)}>Next</Button></div>}
      {history && <section aria-label="Namespace skill revision history"><h2>{history.skill.name} revisions</h2><ul>{history.items.map(item => <li key={item.revision}><Button minimal onClick={() => setPreview(item)}>View revision {item.revision}</Button> · {item.createdBy}</li>)}</ul>{preview && <pre className="studio-skill-revision-content">{preview.bundle.content}</pre>}<Button onClick={() => { setHistory(null); setPreview(null); }}>Close history</Button></section>}
    </>}
  </section>;
}
