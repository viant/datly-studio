import React, { useEffect, useState } from 'react';
import { Button, Callout, Collapse, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Switch, Tag, TextArea } from '@blueprintjs/core';

const emptyDraft = () => ({ name: '', driver: 'sqlite', dsnTemplate: '', secretRef: '', description: '', options: '{}' });

// ConnectorDialog owns only client-side draft state. Creation is delegated to
// the named JSX Studio SDK method, which derives the owner from the verified
// server principal.
export function ConnectorDialog({ api, isOpen, onClose, onCreated, connector = null }) {
  const [draft, setDraft] = useState(emptyDraft);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [conflict, setConflict] = useState(false);
  const [advanced, setAdvanced] = useState(false);
  const [clearSecret, setClearSecret] = useState(false);
  const editing = Boolean(connector?.name);

  useEffect(() => {
    if (!isOpen) return;
    setDraft(connector ? {
      name: connector.name, driver: connector.driver, dsnTemplate: '', secretRef: '', description: connector.description || '', options: JSON.stringify(connector.options || {}, null, 2),
    } : emptyDraft());
    const optionsConfigured = connector?.options && JSON.stringify(connector.options) !== '{}';
    setError(''); setConflict(false); setClearSecret(false); setAdvanced(Boolean(connector?.secretConfigured || optionsConfigured));
  }, [isOpen, connector]);

  const update = (name) => (event) => setDraft((current) => ({ ...current, [name]: event.target.value }));
  const submit = async (event) => {
    event.preventDefault();
    const name = draft.name.trim();
    if (!/^[a-z][a-z0-9_-]{0,127}$/i.test(name)) {
      setError('Use a connector name beginning with a letter; letters, numbers, underscores, and dashes are allowed.');
      return;
    }
    let options;
    try { options = JSON.parse(draft.options || '{}'); }
    catch { setError('Options must be valid JSON.'); return; }
    setSaving(true); setError('');
    try {
      const input = { driver: draft.driver, description: draft.description.trim(), options };
      if (!editing || draft.dsnTemplate.trim()) input.dsnTemplate = draft.dsnTemplate.trim();
      if (!editing || draft.secretRef.trim() || clearSecret) input.secretRef = clearSecret ? '' : draft.secretRef.trim();
      const saved = editing
        ? await api.updateConnector(connector.name, { ...input, etag: connector.etag })
        : await api.createConnector({ name, ...input });
      await onCreated(saved);
      onClose();
    } catch (cause) { setConflict(cause?.code === 'conflict'); setError(cause.message); }
    finally { setSaving(false); }
  };

  const closeAndRefresh = async () => { setSaving(true); try { await onCreated(); onClose(); } finally { setSaving(false); } };
  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title={editing ? `Edit ${connector.name}` : 'New connector'} icon="database" canOutsideClickClose={!saving}>
    <form onSubmit={submit}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">{editing ? 'Update connection configuration through the Studio SDK. Connection-affecting changes require a fresh probe and activation.' : 'Create a data source for Studio readers. Ownership is assigned by the signed-in server principal.'}</p>
        {editing && <div className="studio-validation-revision"><span>Current revision {connector.etag}</span><Tag minimal intent={connector.status === 'active' ? 'success' : 'warning'}>{connector.status}</Tag></div>}
        {editing && connector.status === 'active' && <Callout intent="warning" style={{ marginTop: 14 }}>Changing provider, DSN, secret, or provider options moves this connector to draft and clears its old probe result. Test and activate it again before it can build a reader.</Callout>}
        {conflict ? <Callout intent="warning" role="alert" style={{ marginTop: 14 }}>This connector changed elsewhere. No update was applied. <Button type="button" small minimal intent="warning" icon="refresh" loading={saving} onClick={closeAndRefresh}>Close and refresh catalog</Button></Callout> : error && <Callout intent="danger" role="alert" style={{ marginTop: 14 }}>{error}</Callout>}
        <div className="studio-form-grid">
          <FormGroup label="Name" labelFor="connector-name" helperText="Stable identifier used by components and generated readers." required>
            <InputGroup id="connector-name" value={draft.name} onChange={update('name')} placeholder="reporting_mysql" autoFocus disabled={saving || editing} />
          </FormGroup>
          <FormGroup label="Provider" labelFor="connector-driver" required>
            <HTMLSelect id="connector-driver" value={draft.driver} onChange={update('driver')} fill disabled={saving}>
              <option value="sqlite">SQLite</option><option value="mysql">MySQL</option><option value="postgres">PostgreSQL</option><option value="bigquery">BigQuery</option>
            </HTMLSelect>
          </FormGroup>
        </div>
        <FormGroup label={editing ? 'New DSN template' : 'DSN template'} labelFor="connector-dsn" helperText={editing ? 'Leave blank to keep the existing server-held DSN. Existing connection material is never returned to this browser.' : 'Use a server-side secret reference for credentials; do not paste credentials into the browser.'}>
          <InputGroup id="connector-dsn" type="password" value={draft.dsnTemplate} onChange={update('dsnTemplate')} placeholder={editing && connector.dsnConfigured ? 'DSN configured on server' : 'Provider connection template'} disabled={saving} />
        </FormGroup>
        <FormGroup label="Description" labelFor="connector-description">
          <TextArea id="connector-description" value={draft.description} onChange={update('description')} fill rows={2} disabled={saving} />
        </FormGroup>
        <Button type="button" minimal small icon={advanced ? 'chevron-down' : 'chevron-right'} onClick={() => setAdvanced((value) => !value)} aria-expanded={advanced}>Advanced provider settings</Button>
        <Collapse isOpen={advanced}>
          <div className="studio-advanced-fields">
            <FormGroup label={editing ? 'New secret reference' : 'Secret reference'} labelFor="connector-secret" helperText={editing && connector.secretConfigured ? 'A server-held secret reference is configured. Leave blank to keep it.' : 'Optional Scy resource URL and key; secret material never returns to the browser.'}>
              <InputGroup id="connector-secret" value={draft.secretRef} onChange={update('secretRef')} placeholder={editing && connector.secretConfigured ? 'Secret reference configured on server' : 'gs://bucket/connection.json|key'} disabled={saving || clearSecret} />
            </FormGroup>
            {editing && connector.secretConfigured && <Switch checked={clearSecret} label="Remove the existing secret reference" onChange={(event) => setClearSecret(event.target.checked)} disabled={saving}/>}
            <FormGroup label="Provider options" labelFor="connector-options" helperText="Optional JSON passed to the host provider.">
              <TextArea id="connector-options" value={draft.options} onChange={update('options')} fill rows={3} className="studio-code-input" disabled={saving} />
            </FormGroup>
          </div>
        </Collapse>
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving}>{editing ? 'Save connector' : 'Create connector'}</Button></>} />
    </form>
  </Dialog>;
}
