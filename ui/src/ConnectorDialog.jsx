import React, { useEffect, useState } from 'react';
import { parseDocument } from 'yaml';
import { Button, Callout, Collapse, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Switch, Tag, TextArea } from '@blueprintjs/core';

const splitSecretReference = (reference = '') => {
  const separator = reference.indexOf('|');
  return separator < 0 ? { url: reference, key: '' } : { url: reference.slice(0, separator), key: reference.slice(separator + 1) };
};

const emptyDraft = () => ({ name: '', driver: 'sqlite', dsnTemplate: '', secretURL: '', secretKey: '', description: '', options: '{}' });

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
  const [yamlOpen, setYamlOpen] = useState(false);
  const [yamlText, setYamlText] = useState('');
  const editing = Boolean(connector?.name);
  const currentSecret = splitSecretReference(connector?.secretRefDisplay || '');

  useEffect(() => {
    if (!isOpen) return;
    setDraft(connector ? {
      name: connector.name, driver: connector.driver, dsnTemplate: '', secretURL: '', secretKey: '', description: connector.description || '', options: JSON.stringify(connector.options || {}, null, 2),
    } : emptyDraft());
    const optionsConfigured = connector?.options && JSON.stringify(connector.options) !== '{}';
    setError(''); setConflict(false); setClearSecret(false); setYamlOpen(false); setYamlText(''); setAdvanced(Boolean(connector?.secretConfigured || optionsConfigured));
  }, [isOpen, connector]);

  const update = (name) => (event) => setDraft((current) => ({ ...current, [name]: event.target.value }));
  const applyYAML = () => {
    try {
      const document = parseDocument(yamlText, { uniqueKeys: true });
      if (document.errors.length) throw new Error(document.errors[0].message);
      const value = document.toJS();
      if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Connector YAML must be an object.');
      const fields = Object.keys(value);
      if (!fields.length || fields.some((field) => !['Driver', 'DSN', 'Secret'].includes(field))) throw new Error('Connector YAML supports Driver, DSN, and Secret only.');
      const patch = {};
      if (Object.hasOwn(value, 'Driver')) {
        if (!['sqlite', 'mysql', 'postgres', 'bigquery', 'oracle'].includes(value.Driver)) throw new Error('Choose a supported connector Driver.');
        patch.driver = value.Driver;
      }
      if (Object.hasOwn(value, 'DSN')) {
        if (typeof value.DSN !== 'string') throw new Error('DSN must be a string.');
        patch.dsnTemplate = value.DSN;
      }
      if (Object.hasOwn(value, 'Secret')) {
        if (value.Secret === null) {
          patch.secretURL = ''; patch.secretKey = '';
        } else {
          const secret = value.Secret;
          if (!secret || typeof secret !== 'object' || Array.isArray(secret) || Object.keys(secret).some((field) => !['URL', 'Key'].includes(field))) throw new Error('Secret must contain URL and optional Key.');
          if (typeof secret.URL !== 'string' || !secret.URL.trim() || (Object.hasOwn(secret, 'Key') && typeof secret.Key !== 'string')) throw new Error('Secret requires a URL string and an optional Key string.');
          patch.secretURL = secret.URL; patch.secretKey = secret.Key || '';
        }
      }
      setDraft((current) => ({ ...current, ...patch }));
      if (Object.hasOwn(value, 'Secret')) { setClearSecret(editing && value.Secret === null); setAdvanced(true); }
      setError(''); setConflict(false);
    } catch (cause) { setError(cause.message); }
  };

  const submit = async (event) => {
    event.preventDefault();
    const name = draft.name.trim();
    if (!/^[a-z][a-z0-9_-]{0,127}$/i.test(name)) {
      setError('Use a connector name beginning with a letter; letters, numbers, underscores, and dashes are allowed.');
      return;
    }
    const secretURL = draft.secretURL.trim(), secretKey = draft.secretKey.trim();
    if (!clearSecret && secretKey && !secretURL) { setError('Enter the secret URL together with its key reference.'); return; }
    const secretRef = secretURL ? `${secretURL}${secretKey ? `|${secretKey}` : ''}` : '';
    let options;
    try { options = JSON.parse(draft.options || '{}'); }
    catch { setError('Options must be valid JSON.'); return; }
    setSaving(true); setError('');
    try {
      const input = { driver: draft.driver, description: draft.description.trim(), options };
      if (!editing || draft.dsnTemplate.trim()) input.dsnTemplate = draft.dsnTemplate.trim();
      if (!editing || secretRef || clearSecret) input.secretRef = clearSecret ? '' : secretRef;
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
        {editing && connector.status === 'active' && <Callout intent="warning" className="studio-connector-dialog__notice">Changing provider, DSN, secret, or provider options moves this connector to draft and clears its old probe result. Test and activate it again before it can build a reader.</Callout>}
        {conflict ? <Callout intent="warning" role="alert" className="studio-connector-dialog__notice">This connector changed elsewhere. No update was applied. <Button type="button" small minimal intent="warning" icon="refresh" loading={saving} onClick={closeAndRefresh}>Close and refresh catalog</Button></Callout> : error && <Callout intent="danger" role="alert" className="studio-connector-dialog__notice">{error}</Callout>}
        <Button type="button" minimal small icon="import" onClick={() => setYamlOpen((value) => !value)} aria-expanded={yamlOpen}>Paste YAML configuration</Button>
        <Collapse isOpen={yamlOpen}>
          <FormGroup label="Connector YAML" labelFor="connector-yaml" helperText="Apply Driver, DSN, and Secret.URL/Key to the fields below. Omitted fields remain unchanged; Secret: null explicitly removes the reference.">
            <TextArea id="connector-yaml" value={yamlText} onChange={(event) => setYamlText(event.target.value)} fill rows={8} className="studio-code-input" disabled={saving} />
          </FormGroup>
          <Button type="button" small onClick={applyYAML} disabled={saving || !yamlText.trim()}>Apply YAML</Button>
        </Collapse>
        <div className="studio-form-grid">
          <FormGroup label="Name" labelFor="connector-name" helperText="Stable identifier used by components and generated readers." required>
            <InputGroup id="connector-name" value={draft.name} onChange={update('name')} placeholder="reporting_mysql" autoFocus disabled={saving || editing} />
          </FormGroup>
          <FormGroup label="Provider" labelFor="connector-driver" required>
            <HTMLSelect id="connector-driver" value={draft.driver} onChange={update('driver')} fill disabled={saving}>
              <option value="sqlite">SQLite</option><option value="mysql">MySQL</option><option value="postgres">PostgreSQL</option><option value="bigquery">BigQuery</option><option value="oracle">Oracle</option>
            </HTMLSelect>
          </FormGroup>
        </div>
        {editing && connector.dsnDisplay && <FormGroup label="Current DSN template" labelFor="connector-current-dsn" helperText="Saved template; embedded credentials are masked. Secret values are not resolved.">
          <InputGroup id="connector-current-dsn" value={connector.dsnDisplay} readOnly />
        </FormGroup>}
        {editing && connector.secretRefDisplay && <div className="studio-form-grid">
          <FormGroup label="Current secret URL" labelFor="connector-current-secret-url">
            <InputGroup id="connector-current-secret-url" value={currentSecret.url} readOnly />
          </FormGroup>
          <FormGroup label="Current secret key" labelFor="connector-current-secret-key">
            <InputGroup id="connector-current-secret-key" value={currentSecret.key} readOnly />
          </FormGroup>
        </div>}
        <FormGroup label={editing ? 'New DSN template' : 'DSN template'} labelFor="connector-dsn" helperText={editing ? 'Leave blank to keep the existing server-held DSN. The current template is shown above.' : 'Use a server-side secret reference for credentials; do not paste credentials into the browser.'}>
          <InputGroup id="connector-dsn" type="text" value={draft.dsnTemplate} onChange={update('dsnTemplate')} placeholder={editing && connector.dsnConfigured ? 'DSN configured on server' : 'Provider connection template'} disabled={saving} />
        </FormGroup>
        <FormGroup label="Description" labelFor="connector-description">
          <TextArea id="connector-description" value={draft.description} onChange={update('description')} fill rows={2} disabled={saving} />
        </FormGroup>
        <Button type="button" minimal small icon={advanced ? 'chevron-down' : 'chevron-right'} onClick={() => setAdvanced((value) => !value)} aria-expanded={advanced}>Advanced provider settings</Button>
        <Collapse isOpen={advanced}>
          <div className="studio-advanced-fields">
            <FormGroup label={editing ? 'New secret URL' : 'Secret URL'} labelFor="connector-secret-url" helperText={editing && connector.secretConfigured ? 'Leave both fields blank to keep the server-held reference.' : 'Optional server-side SCY resource URL.'}>
              <InputGroup id="connector-secret-url" value={draft.secretURL} onChange={update('secretURL')} placeholder="file:///config/mysql-studio.enc" disabled={saving || clearSecret} />
            </FormGroup>
            <FormGroup label={editing ? 'New secret key' : 'Secret key'} labelFor="connector-secret-key" helperText="Optional SCY key reference.">
              <InputGroup id="connector-secret-key" value={draft.secretKey} onChange={update('secretKey')} placeholder="blowfish://default" disabled={saving || clearSecret} />
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
