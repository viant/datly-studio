import React, { useEffect, useState } from 'react';
import { Alert, Button, Callout, FormGroup, HTMLSelect, HTMLTable, InputGroup, Spinner, Tab, Tabs, Tag } from '@blueprintjs/core';
import { SecurityWorkspace } from './SecurityWorkspace.jsx';
import { ResourceAccessEditor } from './ResourceAccessEditor.jsx';
import { defaultActionsByKind } from './resourceAccessActions.js';
import { liveACLReviewURL } from './aclReviewURL.js';

export function SecurityCenter({ api, kinds, actionsByKind, choices }) {
  return <div className="studio-workspace">
    <Tabs id="studio-security-sections" defaultSelectedTabId="permissions" renderActiveTabPanelOnly={false}>
      <Tab id="permissions" title="Permissions" panel={<PermissionsWorkspace api={api} kinds={kinds} actionsByKind={actionsByKind} choices={choices}/>}/>
      <Tab id="predicates" title="Authorization predicates" panel={<SecurityWorkspace api={api}/>}/>
    </Tabs>
  </div>;
}

// Embeddings may add kinds, but must provide each kind's action contract.
export function PermissionsWorkspace({ api, kinds = Object.keys(defaultActionsByKind), actionsByKind = defaultActionsByKind, choices = {} }) {
  const [draft, setDraft] = useState({ kind: kinds[0], id: '', version: '1', tenant: '' });
  const [resource, setResource] = useState(null);
  const [selection, setSelection] = useState(null);
  const [editorEpoch, setEditorEpoch] = useState(0);
  const [kind, setKind] = useState('');
  const [access, setAccess] = useState('');
  const [query, setQuery] = useState('');
  const [offset, setOffset] = useState(0);
  const [catalog, setCatalog] = useState({items:[],hasMore:false});
  const [selectedVersions, setSelectedVersions] = useState({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  const [editorState, setEditorState] = useState({dirty:false,saving:false});
  const [pending, setPending] = useState(null);
  const select = item => { setSelection(item); setResource(item.resource); setEditorEpoch(value=>value+1); };
  const choose = item => { if (editorState.saving) return; if (editorState.dirty) setPending(item); else select(item); };
  useEffect(() => {
    if (!api.listAccessResources) return;
    let cancelled=false;
    setLoading(true);setError('');setCatalog({items:[],hasMore:false});
    api.listAccessResources({kind,access,query,limit:10,offset}).then(page=>{if(!cancelled)setCatalog(page);})
      .catch(cause=>{if(!cancelled)setError(cause.message);}).finally(()=>{if(!cancelled)setLoading(false);});
    return()=>{cancelled=true;};
  },[api,kind,access,query,offset,reload]);
  const actions = resource ? actionsByKind[resource.kind] : null;
  const reviewURL = resource && actions && selection?.explicitAcl !== false ? liveACLReviewURL(resource, import.meta.env.BASE_URL) : null;
  const field = name => ({ value: draft[name], onChange: event => setDraft({ ...draft, [name]: event.target.value }) });
  return <section aria-label="Resource permissions">
    <div className="studio-page-heading-row"><h1 className="studio-page-heading">Permissions</h1>{reviewURL && <a className="bp6-button" href={reviewURL} target="_blank" rel="noopener noreferrer">Review current permissions</a>}</div>

    {api.listAccessResources ? <>
      <div className="studio-permission-filters">
        <FormGroup label="Resource type" labelFor="permissions-filter-kind"><HTMLSelect id="permissions-filter-kind" value={kind} onChange={event=>{setKind(event.target.value);setOffset(0);}}><option value="">All types</option>{kinds.map(value=><option key={value} value={value}>{value === 'component' ? 'Components' : value === 'skill' ? 'Skills' : value}</option>)}</HTMLSelect></FormGroup>
        <FormGroup label="Access" labelFor="permissions-filter-access"><HTMLSelect id="permissions-filter-access" value={access} onChange={event=>{setAccess(event.target.value);setOffset(0);}}><option value="">All access</option><option value="public">Public</option><option value="protected">Protected</option></HTMLSelect></FormGroup>
        <FormGroup label="Find a resource" labelFor="permissions-search"><InputGroup id="permissions-search" type="search" placeholder="Name or ID" leftIcon="search" value={query} onChange={event=>{setQuery(event.target.value);setOffset(0);}}/></FormGroup>
      </div>
      {error ? <Callout intent="danger" role="alert" title="Resource list unavailable">{error}<Button onClick={()=>setReload(value=>value+1)}>Retry</Button></Callout> : loading ? <div role="status"><Spinner size={24}/>Loading resources…</div> : <>
        <div className="studio-permission-table"><HTMLTable striped aria-label="Permission resources"><thead><tr><th>Resource</th><th>Type</th><th>Tenant / version</th><th>Permissions</th></tr></thead><tbody>{catalog.items.map(original=>{const item={...original,...(original.versions?.find(version=>JSON.stringify(version.resource)===selectedVersions[`${original.kind}:${original.id}`]) || {})}; return <tr key={JSON.stringify([item.kind,item.id,item.resource])}><td><Button minimal disabled={editorState.saving} onClick={()=>choose(item)}>{item.name}</Button><div><code>{item.id}</code></div>{item.ownerName && item.kind==='skill' && <small>Component: {item.ownerName}</small>}</td><td>{item.kind}</td><td>{item.resource.tenant}{original.versions?.length>1?<HTMLSelect aria-label={`${item.name} version`} value={JSON.stringify(item.resource)} onChange={event=>setSelectedVersions(current=>({...current,[`${item.kind}:${item.id}`]:event.target.value}))}>{original.versions.map(version=><option key={JSON.stringify(version.resource)} value={JSON.stringify(version.resource)}>v{version.resource.version}{version.published?' · Published':''}{version.resource.tenant?` · ${version.resource.tenant}`:''}</option>)}</HTMLSelect>:<div>v{item.resource.version}</div>}</td><td><Tag minimal intent={item.public?'success':'warning'}>{item.public?'Public':'Protected'}{item.inherited?' · inherited':''}</Tag><div>{item.explicitAcl ? 'Explicit ACL' : item.public ? 'No explicit ACL' : 'Component authorization'}</div></td></tr>;})}</tbody></HTMLTable></div>
        {!catalog.items.length && <p role="status">No resources match these filters.</p>}
        <div className="studio-permission-pagination"><Button disabled={!offset || loading} onClick={()=>setOffset(value=>Math.max(0,value-10))}>Previous</Button><span>Page {Math.floor(offset/10)+1}</span><Button disabled={!catalog.hasMore || loading} onClick={()=>setOffset(value=>value+10)}>Next</Button></div>
      </>}
    </> : <form className="studio-form-grid" onSubmit={event => { event.preventDefault(); choose({name:draft.id.trim(),resource:{ ...draft, id: draft.id.trim(), tenant: draft.tenant.trim(), version: draft.version.trim() }}); }}>
      <FormGroup label="Resource type" labelFor="permissions-kind"><HTMLSelect id="permissions-kind" {...field('kind')}>{kinds.map(kind => <option key={kind}>{kind}</option>)}</HTMLSelect></FormGroup>
      <FormGroup label="Resource ID" labelFor="permissions-id"><InputGroup id="permissions-id" required {...field('id')}/></FormGroup>
      <FormGroup label="Tenant" labelFor="permissions-tenant"><InputGroup id="permissions-tenant" required {...field('tenant')}/></FormGroup>
      <FormGroup label="Version" labelFor="permissions-version"><InputGroup id="permissions-version" required {...field('version')}/></FormGroup>
      <div><Button type="submit" intent="primary" icon="lock" disabled={!draft.id.trim() || !draft.tenant.trim() || !draft.version.trim()}>Load permissions</Button></div>
    </form>}
    {selection?.inherited && selection?.explicitAcl !== false && <Callout compact>Skill {selection.name} inherits permissions from {selection.ownerName}. Changes below apply to the owning component.</Callout>}
    <Alert isOpen={Boolean(pending)} cancelButtonText="Keep editing" confirmButtonText="Discard changes" intent="warning" onCancel={()=>setPending(null)} onConfirm={()=>{select(pending);setPending(null);}}><p>Discard unsaved permission changes and open another resource?</p></Alert>
    {resource && selection?.explicitAcl === false ? <Callout icon={selection.public?'unlock':'lock'} intent={selection.public?'success':'warning'} title={`${selection.name} · ${selection.public?'Public':'Protected'}`}>{selection.public ? 'No explicit ACL is defined for this resource.' : 'This resource uses component authorization without a standalone policy at this version.'}</Callout> : resource ? Array.isArray(actions) && actions.length ? <ResourceAccessEditor key={editorEpoch} api={api} resource={resource} choices={choices} actions={actions} displayName={selection?.name} onStateChange={setEditorState}/> : <Callout intent="warning" title="Action contract required">Configure the permission actions for resource type <code>{resource.kind}</code> before opening its policy.</Callout> : <Callout icon="lock" title="Choose a resource">Select a resource to view its access settings.</Callout>}
  </section>;
}
