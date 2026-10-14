import { NamespaceSkills } from './NamespaceSkills.jsx';
import React, { useCallback, useEffect, useState } from 'react';
import { Button, Callout, FormGroup, HTMLSelect, HTMLTable, InputGroup, Spinner, Tab, Tabs } from '@blueprintjs/core';

// Authoring reads the authorized catalog, not the live publication runtime.
export async function componentSkills(api) {
  const components = [];
  for (let offset = 0; ; offset += 100) {
    const page = await api.listComponents({ limit: 100, offset });
    const rows = page?.items ?? [];
    components.push(...rows);
    if (rows.length < 100) break;
  }
  const skills = [], failures = [];
  // Bound requests: each resource snapshot compiles an exact-version reader.
  for (const component of components) {
    try {
      const versionNo = component.currentDraftVersion || (await api.listVersions(component.id, { limit: 1 }))?.items?.[0]?.versionNo;
      if (!versionNo) continue;
      const snapshot = await api.getResources(component.id, versionNo);
      for (const skill of snapshot.skills ?? []) skills.push({ ...skill, component, versionNo });
    } catch (cause) {
      if (cause.name === 'AbortError') throw cause;
      failures.push(`${component.title}: ${cause.message}`);
    }
  }
  return { components, skills, failures };
}

export function SkillsWorkspace({ api, namespace, subject, renderNamespaceSkills, onOpenComponent, onDirtyChange }) {
  const [ownership, setOwnership] = useState('namespace');
  const [editing, setEditing] = useState(false);
  const changed = useCallback((dirty, state = {}) => { const blocked = Boolean(dirty || state.busy); setEditing(blocked); onDirtyChange?.(blocked); }, [onDirtyChange]);
  useEffect(() => () => onDirtyChange?.(false), [onDirtyChange]);
  return <main className="studio-workspace studio-runtime-workspace">
    <div className="studio-runtime-header"><div><h1 className="studio-page-heading">Skills</h1><p className="studio-page-description">Manage reusable skills in {namespace?.title || namespace?.name || 'the selected namespace'}.</p></div></div>
    <Tabs id="skill-ownership" selectedTabId={ownership} onChange={value => { if (!editing) setOwnership(value); }} renderActiveTabPanelOnly>
      <Tab id="namespace" title="Namespace-owned" disabled={editing && ownership !== 'namespace'} panel={renderNamespaceSkills ? renderNamespaceSkills({ api, namespace, subject, onDirtyChange: changed }) : <NamespaceSkills api={api} namespace={namespace} onDirtyChange={changed}/>}/>
      <Tab id="component" title="Component-owned" disabled={editing && ownership !== 'component'} panel={<ComponentSkills api={api} onOpenComponent={onOpenComponent}/>}/>
    </Tabs>
  </main>;
}

function ComponentSkills({ api, onOpenComponent }) {
  const [catalog, setCatalog] = useState({ components: [], skills: [], failures: [] });
  const [loading, setLoading] = useState(true), [error, setError] = useState('');
  const [refresh, setRefresh] = useState(0), [ownerID, setOwnerID] = useState(''), [query, setQuery] = useState('');
  useEffect(() => {
    let cancelled = false;
    setLoading(true); setError('');
    componentSkills(api).then(value => { if (!cancelled) setCatalog(value); })
      .catch(cause => { if (!cancelled) setError(cause.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [api, refresh]);
  const open = (component, versionNo, create = false, resourceSkillId = '') => onOpenComponent?.({ ...component, versionNo, resourceSkillId, openResources: true, resourceAction: create ? 'newSkill' : '' });
  const owner = catalog.components.find(item => item.id === ownerID);
  const rows = catalog.skills.filter(item => `${item.skillRoot} ${item.component.title}`.toLowerCase().includes(query.toLowerCase()));
  return <section aria-label="Component-owned skills">
    <p>These skills belong to a component version. Publication makes them available to MCP hosts.</p>
    <div className="studio-skill-toolbar">
      <FormGroup label="Component" labelFor="skill-component"><HTMLSelect id="skill-component" value={ownerID} onChange={event => setOwnerID(event.target.value)} disabled={loading}><option value="">Choose a component</option>{catalog.components.map(item => <option key={item.id} value={item.id}>{item.title}</option>)}</HTMLSelect></FormGroup>
      <Button icon="add" intent="primary" disabled={!owner || loading} onClick={() => open(owner, owner.currentDraftVersion, true)}>New component skill</Button>
      <Button icon="refresh" disabled={loading} onClick={() => setRefresh(value => value + 1)}>Refresh</Button>
      <InputGroup aria-label="Search component skills" placeholder="Find skill or component" value={query} onChange={event => setQuery(event.target.value)}/>
    </div>
    {error && <Callout intent="danger" role="alert" title="Component skills unavailable">{error}</Callout>}
    {catalog.failures.length > 0 && <Callout intent="warning" role="alert" title="Some component skills could not be read">{catalog.failures.map(message => <p key={message}>{message}</p>)}</Callout>}
    {loading ? <Spinner size={22}/> : rows.length ? <div className="studio-table-wrap"><HTMLTable className="studio-data-table" striped><thead><tr><th>Skill</th><th>Component</th><th>Version</th><th>Actions</th></tr></thead><tbody>{rows.map(item => <tr key={`${item.component.id}:${item.versionNo}:${item.skillId}`}><td>{item.skillRoot}</td><td>{item.component.title}</td><td>{item.versionNo}</td><td><Button small icon="edit" onClick={() => open(item.component, item.versionNo, false, item.skillId)}>Edit skill</Button></td></tr>)}</tbody></HTMLTable></div> : !error && <p>No component skills in this namespace. Choose a component to create one.</p>}
  </section>;
}
