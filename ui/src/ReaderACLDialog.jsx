import React, { useEffect, useState } from 'react';
import { Alert, Button, ButtonGroup, Callout, Dialog, DialogBody, DialogFooter, FormGroup, InputGroup, Switch, Tag } from '@blueprintjs/core';

const blank = () => ({ subjectType: 'user', subjectId: '', canView: true, canRun: false, canEdit: false, canPublish: false, canUseDql: false, etag: 0 });
const presets={viewer:{canView:true,canRun:false,canEdit:false,canPublish:false,canUseDql:false},operator:{canView:true,canRun:true,canEdit:false,canPublish:false,canUseDql:false},author:{canView:true,canRun:true,canEdit:true,canPublish:false,canUseDql:false},advanced:{canView:true,canRun:true,canEdit:true,canPublish:false,canUseDql:true},publisher:{canView:true,canRun:true,canEdit:false,canPublish:true,canUseDql:false}};

export function ReaderACLDialog({ isOpen, api, report, onClose }) {
  const [entries, setEntries] = useState([]);
  const [draft, setDraft] = useState(blank);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [revoke, setRevoke] = useState(null);
  const [query,setQuery]=useState('');
  const [conflict,setConflict]=useState(false);
  const load = async () => {
    if (!report?.id) return;
    setLoading(true); setError(''); setConflict(false);
    try { setEntries(await api.listACL(report.id)); }
    catch (cause) { setError(cause.message); }
    finally { setLoading(false); }
  };
  useEffect(() => { if (isOpen) { setDraft(blank()); setRevoke(null); setQuery(''); load(); } }, [isOpen, report?.id]);
  const save = async () => {
    if (!draft.subjectId.trim()) { setError('Subject ID is required.'); return; }
    setSaving(true); setError(''); setConflict(false);
    try {
      await api.upsertACL({ reportId: report.id, ...draft, subjectId: draft.subjectId.trim() });
      setDraft(blank()); await load();
    } catch (cause) { setConflict(cause?.code==='conflict');setError(cause.message); }
    finally { setSaving(false); }
  };
  const remove = async () => {
    if (!revoke) return;
    setSaving(true); setError(''); setConflict(false);
    try { await api.deleteACL(report.id, revoke.subjectType, revoke.subjectId, revoke.etag); setRevoke(null); await load(); }
    catch (cause) { setConflict(cause?.code==='conflict');setError(cause.message); }
    finally { setSaving(false); }
  };
  const choose = (entry) => setDraft({ subjectType: entry.subjectType, subjectId: entry.subjectId, canView: entry.canView, canRun: entry.canRun, canEdit: entry.canEdit, canPublish: entry.canPublish, canUseDql: entry.canUseDql, etag:entry.etag||0 });
  const setCapability=(name,value)=>setDraft((current)=>normalizeCapabilities({...current,[name]:value},name));
  const filtered=entries.filter((entry)=>`${entry.subjectId} ${capabilityNames(entry).join(' ')}`.toLowerCase().includes(query.trim().toLowerCase()));

  return <Dialog className="studio-connector-dialog studio-acl-dialog" isOpen={isOpen} onClose={onClose} title="Reader permissions" icon="lock" canOutsideClickClose={!saving}>
    <DialogBody className="studio-connector-dialog-body">
      <p className="studio-dialog-lead">Component owners grant explicit access to one verified JWT subject. These capabilities apply to Studio authoring and dynamic reader use; ownership itself remains implicit and is never editable here.</p>
      {error&&!conflict && <Callout intent="danger" role="alert">{error}</Callout>}
      {conflict&&<Callout intent="warning" title="Permission policy changed elsewhere" role="alert">No access change was applied. Reload the current grants before reviewing and retrying.<Button small minimal intent="warning" icon="refresh" onClick={load}>Reload permissions</Button></Callout>}
      <section className="studio-acl-list"><div className="studio-panel-heading"><div><h3>Current access</h3><p className="studio-muted">Delegated JWT subjects only; owner authority is implicit.</p></div>{entries.length>0&&<Button small icon="add" onClick={()=>setDraft(blank())}>New grant</Button>}</div>{entries.length>6&&<InputGroup leftIcon="search" aria-label="Search permission grants" placeholder="Find subject or capability" value={query} onChange={(event)=>setQuery(event.target.value)} rightElement={query?<Button minimal icon="cross" aria-label="Clear permission search" onClick={()=>setQuery('')}/>:undefined}/>} {loading && <span className="studio-muted">Loading permissions…</span>}{!loading && entries.length === 0 && <Callout intent="primary">No delegated access. The component owner retains full control.</Callout>}{!loading&&entries.length>0&&filtered.length===0&&<div className="studio-empty-compact">No grants match this search.</div>}<div className="studio-acl-entry-list">{filtered.map((entry) => <div key={`${entry.subjectType}:${entry.subjectId}`} className={`studio-acl-entry ${draft.subjectId===entry.subjectId?'selected':''}`}><button type="button" onClick={() => choose(entry)}><strong>{entry.subjectId}</strong><small>JWT subject · revision {entry.etag||'—'}</small></button><div className="studio-acl-tags"><PermissionTags entry={entry}/><Button small minimal intent="danger" icon="trash" aria-label={`Revoke ${entry.subjectId}`} onClick={() => setRevoke(entry)}/></div></div>)}</div></section>
      <section className="studio-acl-form"><div className="studio-panel-heading"><div><h3>{draft.etag?`Edit ${draft.subjectId}`:'Grant access'}</h3><p className="studio-muted">Start with a least-privilege role, then adjust only what this person needs.</p></div></div><div className="studio-acl-presets" aria-label="Permission presets"><ButtonGroup minimal>{Object.entries({viewer:'Viewer',operator:'Operator',author:'SQL author',advanced:'Advanced author',publisher:'Publisher'}).map(([key,label])=><Button type="button" small key={key} onClick={()=>setDraft((current)=>({...current,...presets[key]}))}>{label}</Button>)}</ButtonGroup></div><FormGroup label="JWT subject" labelFor="acl-subject-id" helperText="Exact sub claim from the user’s verified token; immutable after the grant is created." required><InputGroup id="acl-subject-id" value={draft.subjectId} onChange={(event) => setDraft({ ...draft, subjectId: event.target.value })} disabled={saving||draft.etag>0} placeholder="user-subject"/></FormGroup><div className="studio-acl-capability-groups"><fieldset><legend>Data access</legend><Switch checked={draft.canView} label="View component and catalog" onChange={(event)=>setCapability('canView',event.target.checked)}/><Switch checked={draft.canRun} label="Run reader and previews" onChange={(event)=>setCapability('canRun',event.target.checked)}/></fieldset><fieldset><legend>Authoring</legend><Switch checked={draft.canEdit} label="Edit SQL and structured settings" onChange={(event)=>setCapability('canEdit',event.target.checked)}/><Switch checked={draft.canUseDql} label="Use advanced structural DQL" onChange={(event)=>setCapability('canUseDql',event.target.checked)}/></fieldset><fieldset><legend>Release</legend><Switch checked={draft.canPublish} label="Validate and publish runtime" onChange={(event)=>setCapability('canPublish',event.target.checked)}/></fieldset></div><div className="studio-acl-effective"><span>Effective grant</span><PermissionTags entry={draft}/></div><Button intent="primary" icon="floppy-disk" loading={saving} onClick={save}>{draft.etag?'Update grant':'Create grant'}</Button></section>
    </DialogBody>
    <DialogFooter actions={<Button onClick={onClose} disabled={saving}>Close</Button>}/>
    <Alert isOpen={Boolean(revoke)} intent="danger" icon="trash" confirmButtonText="Revoke access" cancelButtonText="Cancel" onCancel={() => setRevoke(null)} onConfirm={remove} canEscapeKeyCancel canOutsideClickCancel>
      Revoke all delegated capabilities for <strong>{revoke?.subjectId}</strong>?
    </Alert>
  </Dialog>;
}

function PermissionTags({ entry }) {
  return <>{entry.canView && <Tag minimal>view</Tag>}{entry.canRun && <Tag minimal>run</Tag>}{entry.canEdit && <Tag minimal>edit</Tag>}{entry.canPublish && <Tag minimal intent="warning">publish</Tag>}{entry.canUseDql && <Tag minimal>dql</Tag>}</>;
}
function capabilityNames(entry){return [['view',entry.canView],['run',entry.canRun],['edit',entry.canEdit],['publish',entry.canPublish],['dql',entry.canUseDql]].filter(([,enabled])=>enabled).map(([name])=>name);}
function normalizeCapabilities(value,changed){const next={...value};if(changed==='canView'&&!next.canView){next.canRun=false;next.canEdit=false;next.canPublish=false;next.canUseDql=false;}if(next.canRun||next.canEdit||next.canPublish||next.canUseDql)next.canView=true;if(next.canUseDql)next.canEdit=true;if(!next.canEdit)next.canUseDql=false;return next;}
