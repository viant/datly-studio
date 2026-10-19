import React, { useEffect, useState } from 'react';
import { Button, Callout, Card, FormGroup, InputGroup, HTMLSelect } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';

export function ReaderCreateForm({ report, onCreate, onCancel, importAction, disabled = false, onDirtyChange, onBusyChange }) {
  const [name, setName] = useState('record');
  const [route, setRoute] = useState(`/v1/studio/readers/${report.slug || report.id}`);
  const [visibility, setVisibility] = useState('public');
  const [sql, setSQL] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const dirty = visibility !== 'public' || sql !== '' || name !== 'record' || route !== `/v1/studio/readers/${report.slug || report.id}`;
  useEffect(() => { onDirtyChange?.(dirty); return () => onDirtyChange?.(false); }, [dirty, onDirtyChange]);
  useEffect(() => { onBusyChange?.(saving); return () => onBusyChange?.(false); }, [saving, onBusyChange]);
  const submit = async (event) => {
    event.preventDefault();
    if (!/^[a-z][a-z0-9_]*$/.test(name) || !route.trim().startsWith('/') || !sql.trim()) {
      setError('Enter a lowercase view identifier, a route starting with /, and source SQL.');
      return;
    }
    setSaving(true); setError('');
    try { await onCreate({ name, route: route.trim(), sql }, visibility); }
    catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };
  return <Card className="studio-card studio-schema-editor" elevation={0}>
    <form onSubmit={submit}>
      <div className="studio-panel-heading"><div><h2>Root SQL</h2><p className="studio-muted">Write the root query using connector {report.defaultConnectorName}. You can add inputs, predicates, and related views after creating the draft.</p></div></div>
      {error && <Callout intent="danger" role="alert">{error}</Callout>}
      <div className="studio-form-grid studio-binding-grid">
        <FormGroup label="Root view identifier" labelFor="new-reader-name" helperText="Lowercase letters, numbers, and underscores." required><InputGroup id="new-reader-name" value={name} onChange={(event) => setName(event.target.value)} disabled={saving || disabled} autoFocus/></FormGroup>
        <FormGroup label="Reader route" labelFor="new-reader-route" required><InputGroup id="new-reader-route" value={route} onChange={(event) => setRoute(event.target.value)} disabled={saving || disabled}/></FormGroup>
      </div>
      <FormGroup label="Component visibility" labelFor="new-reader-visibility" helperText={visibility === 'internal' ? 'Internal components expose neither an HTTP endpoint nor an MCP tool.' : 'HTTP routing becomes available after publication. Authentication and authorization still apply.'}>
        <HTMLSelect id="new-reader-visibility" value={visibility} onChange={(event)=>setVisibility(event.target.value)} disabled={saving || disabled}><option value="public">Public HTTP route</option><option value="internal">Internal component</option></HTMLSelect>
      </FormGroup>
      <FormGroup label="Source SQL" helperText="For example: SELECT id, name FROM customers" required>
        <div className="studio-code-editor"><Editor ariaLabel="New reader SQL source" value={sql} onChange={setSQL} language="sql" height="300px" readOnly={saving || disabled}/></div>
      </FormGroup>
      <div className="studio-schema-actions"><Button type="submit" intent="primary" icon="new-object" loading={saving} disabled={disabled}>Create reader</Button>{!saving && importAction}<Button type="button" onClick={onCancel} disabled={saving || disabled}>Cancel</Button></div>
    </form>
  </Card>;
}
