import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, TextArea, Switch, TagInput } from '@blueprintjs/core';

const emptyDraft = () => ({ name: '', title: '', description: '', status: 'active', visibility: 'private', allowedRoles: [], mcpEnabled: false, mcpPort: '0' });
const namespaceDraft = (value) => value ? { ...emptyDraft(), name: value.name, title: value.title, description: value.description || '', status: value.status, visibility: value.visibility || 'private', allowedRoles: value.allowedRoles || [], mcpEnabled: Boolean(value.mcpEnabled), mcpPort: String(value.mcpPort ?? 0) } : emptyDraft();

export function NamespaceDialog({ api, namespace, isOpen, onClose, onSaved }) {
  const [draft, setDraft] = useState(emptyDraft);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [conflict, setConflict] = useState(false);
  const [etag, setEtag] = useState(0);
  const [roleInput, setRoleInput] = useState('');
  const [canManage, setCanManage] = useState(true);
  const [ownerId, setOwnerId] = useState('');
  const editing = Boolean(namespace);
  const readOnly = editing && !canManage;
  const closeDialog = () => { if (!saving) onClose(); };

  useEffect(() => {
    if (!isOpen) return;
    setDraft(namespaceDraft(namespace));
    setRoleInput(''); setCanManage(namespace?.canManage !== false); setOwnerId(namespace?.ownerId || '');
    setSaving(false);
    setError('');
    setConflict(false);
    setEtag(namespace?.etag||0);
  }, [isOpen, namespace]);

  const submit = async (event) => {
    event.preventDefault();
    if (readOnly) return;
    const name = draft.name.trim();
    const title = draft.title.trim();
    if (!/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$/.test(name) || !title) {
      setError('Enter a title and a lowercase dot-separated namespace such as inventory or delivery.forecasting.');
      return;
    }
    const portValid = /^\d+$/.test(draft.mcpPort) && Number(draft.mcpPort) <= 65535;
    if (draft.mcpEnabled && !portValid) { setError('Enter an MCP port from 0 to 65535.'); return; }
    const settings = { visibility: draft.visibility, allowedRoles: [...new Set([...draft.allowedRoles, ...(roleInput.trim() ? [roleInput.trim()] : [])])], mcpEnabled: draft.mcpEnabled, mcpPort: portValid ? Number(draft.mcpPort) : 0 };
    setSaving(true);
    setError(''); setConflict(false);
    try {
      const saved = editing
        ? await api.updateNamespace(namespace.name, { title, description: draft.description.trim(), status: draft.status, etag, ...settings })
        : await api.createNamespace({ name, title, description: draft.description.trim(), ...settings });
      await onSaved(saved);
      onClose();
    } catch (cause) {
      setConflict(cause?.code==='conflict');setError(cause.message);
    } finally {
      setSaving(false);
    }
  };
  const reload=async()=>{
    if(!namespace?.name)return;
    setSaving(true);setError('');
    try{const current=await api.getNamespace(namespace.name);setDraft(namespaceDraft(current));setRoleInput('');setCanManage(current.canManage !== false);setOwnerId(current.ownerId || '');setEtag(current.etag);setConflict(false);}
    catch(cause){setError(cause.message);}finally{setSaving(false);}
  };

  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={closeDialog} title={readOnly ? `Namespace · ${namespace.name}` : editing ? `Edit namespace · ${namespace.name}` : 'New namespace'} icon="folder-shared" canOutsideClickClose={!saving} canEscapeKeyClose={!saving} isCloseButtonShown={!saving}>
    <form onSubmit={submit} noValidate>
      <DialogBody className="studio-connector-dialog-body">
        {editing && ownerId && <p className="studio-dialog-lead">Owner: {ownerId}</p>}
        {error&&!conflict && <Callout intent="danger" role="alert" style={{ marginBottom: 14 }}>{error}</Callout>}
        {conflict&&<Callout intent="warning" title="Namespace changed elsewhere" role="alert" style={{marginBottom:14}}>Your change was not applied. Reload the current namespace before reviewing and retrying.<Button small minimal intent="warning" icon="refresh" onClick={reload}>Reload namespace</Button></Callout>}
        <FormGroup label="Namespace" labelFor="namespace-name"  required>
          <InputGroup id="namespace-name" value={draft.name} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} placeholder="inventory" autoFocus={!editing} disabled={saving || editing || readOnly}/>
        </FormGroup>
        <FormGroup label="Display title" labelFor="namespace-title" required>
          <InputGroup id="namespace-title" value={draft.title} onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))} placeholder="Inventory" autoFocus={editing} disabled={saving || readOnly}/>
        </FormGroup>
        <FormGroup label="Description" labelFor="namespace-description">
          <TextArea id="namespace-description" value={draft.description} onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} fill rows={2} disabled={saving || readOnly}/>
        </FormGroup>
        <FormGroup label="Visibility" labelFor="namespace-visibility" helperText={draft.visibility === 'private' ? 'Visible to the owner and assigned roles.' : 'Visible to everyone. Resource permissions still apply.'}>
          <HTMLSelect id="namespace-visibility" fill value={draft.visibility} disabled={saving || readOnly} onChange={(event) => setDraft((current) => ({ ...current, visibility: event.target.value }))} options={[{ value: 'private', label: 'Private' }, { value: 'public', label: 'Public' }]}/>
        </FormGroup>
        {draft.visibility === 'private' && <FormGroup label="Viewer roles" labelFor="namespace-roles" helperText="Use exact role names. Viewer roles do not grant namespace management.">
          <TagInput inputProps={{ id: 'namespace-roles', 'aria-label': 'Viewer roles' }} fill values={draft.allowedRoles} disabled={saving || readOnly} placeholder="Add a role" inputValue={roleInput} addOnBlur onInputChange={(event) => setRoleInput(event.target.value)} onAdd={(values) => { setRoleInput(''); setDraft((current) => ({ ...current, allowedRoles: [...new Set([...current.allowedRoles, ...values.map((value) => value.trim()).filter(Boolean)])] })); }} onRemove={(_, index) => setDraft((current) => ({ ...current, allowedRoles: current.allowedRoles.filter((_, position) => position !== index) }))}/>
        </FormGroup>}
        <Switch label="Expose MCP" checked={draft.mcpEnabled} disabled={saving || readOnly} onChange={(event) => setDraft((current) => ({ ...current, mcpEnabled: event.target.checked }))}/>
        <div className="studio-form-grid">
        {draft.mcpEnabled && <FormGroup label="MCP port" labelFor="namespace-mcp-port" helperText="0 assigns an available port.">
          <InputGroup id="namespace-mcp-port" type="number" min="0" max="65535" value={draft.mcpPort} disabled={saving || readOnly} onChange={(event) => setDraft((current) => ({ ...current, mcpPort: event.target.value }))}/>
        </FormGroup>}
        {editing && <FormGroup label="Lifecycle" labelFor="namespace-status" helperText={draft.status === 'archived' ? 'Keeps existing components; prevents new ones.' : undefined}>
          <HTMLSelect id="namespace-status" value={draft.status} onChange={(event) => setDraft((current) => ({ ...current, status: event.target.value }))} fill disabled={saving || readOnly}>
            <option value="active">Active</option>
            <option value="archived">Archived</option>
          </HTMLSelect>
        </FormGroup>}
        </div>
      </DialogBody>
      <DialogFooter actions={<><Button onClick={closeDialog} disabled={saving}>{readOnly ? 'Close' : 'Cancel'}</Button>{!readOnly && <Button type="submit" intent="primary" icon="floppy-disk" loading={saving}>{editing ? 'Save namespace' : 'Create namespace'}</Button>}</>}/>
    </form>
  </Dialog>;
}
