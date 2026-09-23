import React, { useState } from 'react';
import { Button, Callout, FormGroup, HTMLSelect, InputGroup, Tab, Tabs } from '@blueprintjs/core';
import { SecurityWorkspace } from './SecurityWorkspace.jsx';
import { ResourceAccessEditor } from './ResourceAccessEditor.jsx';

export function SecurityCenter({ api }) {
  return <div className="studio-workspace">
    <Tabs id="studio-security-sections" defaultSelectedTabId="permissions" renderActiveTabPanelOnly>
      <Tab id="permissions" title="Permissions" panel={<PermissionsWorkspace api={api}/>}/>
      <Tab id="predicates" title="Authorization predicates" panel={<SecurityWorkspace api={api}/>}/>
    </Tabs>
  </div>;
}

// Resource kinds are configurable so an embedding product can add reports or
// other artifacts without adding their workflows to Studio.
export function PermissionsWorkspace({ api, kinds = ['component', 'skill'], choices = {} }) {
  const [draft, setDraft] = useState({ kind: kinds[0], id: '', version: '1', tenant: '' });
  const [resource, setResource] = useState(null);
  const field = name => ({ value: draft[name], onChange: event => setDraft({ ...draft, [name]: event.target.value }) });
  return <section aria-label="Resource permissions">
    <div className="studio-page-heading-row"><h1 className="studio-page-heading">Permissions</h1><a className="bp6-button" href={`${import.meta.env.BASE_URL}acl-review.html`} target="_blank" rel="noopener noreferrer">Open ACL UX review</a></div>
    <p className="studio-page-description">Manage access to components and skills, with separate rules for each action.</p>
    <form className="studio-form-grid" onSubmit={event => { event.preventDefault(); setResource({ ...draft, id: draft.id.trim(), tenant: draft.tenant.trim(), version: draft.version.trim() }); }}>
      <FormGroup label="Resource type" labelFor="permissions-kind"><HTMLSelect id="permissions-kind" {...field('kind')}>{kinds.map(kind => <option key={kind}>{kind}</option>)}</HTMLSelect></FormGroup>
      <FormGroup label="Resource ID" labelFor="permissions-id"><InputGroup id="permissions-id" required {...field('id')}/></FormGroup>
      <FormGroup label="Tenant" labelFor="permissions-tenant"><InputGroup id="permissions-tenant" required {...field('tenant')}/></FormGroup>
      <FormGroup label="Version" labelFor="permissions-version"><InputGroup id="permissions-version" required {...field('version')}/></FormGroup>
      <div><Button type="submit" intent="primary" icon="lock" disabled={!draft.id.trim() || !draft.tenant.trim() || !draft.version.trim()}>Load permissions</Button></div>
    </form>
    {resource ? <ResourceAccessEditor api={api} resource={resource} choices={choices} actions={['discover', 'describe', resource.kind === 'skill' ? 'retrieve' : 'execute', 'export', 'edit', 'validate', 'publish', 'unpublish', 'viewAccess', 'manageAccess']}/> : <Callout icon="lock" title="Choose a resource">Load a provisioned resource policy to review access. Your authenticated identity determines which policies you can view or change.</Callout>}
  </section>;
}
