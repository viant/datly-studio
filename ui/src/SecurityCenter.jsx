import React, { useState } from 'react';
import { Button, Callout, FormGroup, HTMLSelect, InputGroup, Tab, Tabs } from '@blueprintjs/core';
import { SecurityWorkspace } from './SecurityWorkspace.jsx';
import { ResourceAccessEditor } from './ResourceAccessEditor.jsx';
import { defaultActionsByKind } from './resourceAccessActions.js';

export function SecurityCenter({ api, kinds, actionsByKind, choices }) {
  return <div className="studio-workspace">
    <Tabs id="studio-security-sections" defaultSelectedTabId="permissions" renderActiveTabPanelOnly>
      <Tab id="permissions" title="Permissions" panel={<PermissionsWorkspace api={api} kinds={kinds} actionsByKind={actionsByKind} choices={choices}/>}/>
      <Tab id="predicates" title="Authorization predicates" panel={<SecurityWorkspace api={api}/>}/>
    </Tabs>
  </div>;
}

// Embeddings may add kinds, but must provide each kind's action contract.
export function PermissionsWorkspace({ api, kinds = Object.keys(defaultActionsByKind), actionsByKind = defaultActionsByKind, choices = {} }) {
  const [draft, setDraft] = useState({ kind: kinds[0], id: '', version: '1', tenant: '' });
  const [resource, setResource] = useState(null);
  const actions = resource ? actionsByKind[resource.kind] : null;
  const field = name => ({ value: draft[name], onChange: event => setDraft({ ...draft, [name]: event.target.value }) });
  return <section aria-label="Resource permissions">
    <div className="studio-page-heading-row"><h1 className="studio-page-heading">Permissions</h1><a className="bp6-button" href={`${import.meta.env.BASE_URL}acl-review.html`} target="_blank" rel="noopener noreferrer">Open ACL UX review</a></div>
    <p className="studio-page-description">Manage each resource type with separate rules for its declared actions.</p>
    <form className="studio-form-grid" onSubmit={event => { event.preventDefault(); setResource({ ...draft, id: draft.id.trim(), tenant: draft.tenant.trim(), version: draft.version.trim() }); }}>
      <FormGroup label="Resource type" labelFor="permissions-kind"><HTMLSelect id="permissions-kind" {...field('kind')}>{kinds.map(kind => <option key={kind}>{kind}</option>)}</HTMLSelect></FormGroup>
      <FormGroup label="Resource ID" labelFor="permissions-id"><InputGroup id="permissions-id" required {...field('id')}/></FormGroup>
      <FormGroup label="Tenant" labelFor="permissions-tenant"><InputGroup id="permissions-tenant" required {...field('tenant')}/></FormGroup>
      <FormGroup label="Version" labelFor="permissions-version"><InputGroup id="permissions-version" required {...field('version')}/></FormGroup>
      <div><Button type="submit" intent="primary" icon="lock" disabled={!draft.id.trim() || !draft.tenant.trim() || !draft.version.trim()}>Load permissions</Button></div>
    </form>
    {resource ? Array.isArray(actions) && actions.length ? <ResourceAccessEditor api={api} resource={resource} choices={choices} actions={actions}/> : <Callout intent="warning" title="Action contract required">Configure the permission actions for resource type <code>{resource.kind}</code> before opening its policy.</Callout> : <Callout icon="lock" title="Choose a resource">Load a provisioned resource policy to review access. Your authenticated identity determines which policies you can view or change.</Callout>}
  </section>;
}
