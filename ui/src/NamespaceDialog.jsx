import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, TextArea } from '@blueprintjs/core';

const emptyDraft = () => ({ name: '', title: '', description: '', status: 'active' });

export function NamespaceDialog({ api, namespace, isOpen, onClose, onSaved }) {
  const [draft, setDraft] = useState(emptyDraft);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [conflict, setConflict] = useState(false);
  const [etag, setEtag] = useState(0);
  const editing = Boolean(namespace);

  useEffect(() => {
    if (!isOpen) return;
    setDraft(namespace ? {
      name: namespace.name,
      title: namespace.title,
      description: namespace.description || '',
      status: namespace.status,
    } : emptyDraft());
    setSaving(false);
    setError('');
    setConflict(false);
    setEtag(namespace?.etag||0);
  }, [isOpen, namespace]);

  const submit = async (event) => {
    event.preventDefault();
    const name = draft.name.trim();
    const title = draft.title.trim();
    if (!/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$/.test(name) || !title) {
      setError('Enter a title and a lowercase dot-separated namespace such as inventory or delivery.forecasting.');
      return;
    }
    setSaving(true);
    setError(''); setConflict(false);
    try {
      const saved = editing
        ? await api.updateNamespace(namespace.name, { title, description: draft.description.trim(), status: draft.status, etag })
        : await api.createNamespace({ name, title, description: draft.description.trim() });
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
    try{const current=await api.getNamespace(namespace.name);setDraft({name:current.name,title:current.title,description:current.description||'',status:current.status});setEtag(current.etag);setConflict(false);}
    catch(cause){setError(cause.message);}finally{setSaving(false);}
  };

  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title={editing ? `Edit namespace · ${namespace.name}` : 'New namespace'} icon="folder-shared" canOutsideClickClose={!saving}>
    <form onSubmit={submit}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">Namespaces organize production components without changing their server-owned Go package identity or connector bindings.</p>
        {error&&!conflict && <Callout intent="danger" role="alert" style={{ marginBottom: 14 }}>{error}</Callout>}
        {conflict&&<Callout intent="warning" title="Namespace changed elsewhere" role="alert" style={{marginBottom:14}}>Your change was not applied. Reload the current namespace before reviewing and retrying.<Button small minimal intent="warning" icon="refresh" onClick={reload}>Reload namespace</Button></Callout>}
        <FormGroup label="Namespace" labelFor="namespace-name" helperText="Immutable lowercase catalog identity; dot-separated segments are supported." required>
          <InputGroup id="namespace-name" value={draft.name} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} placeholder="inventory" autoFocus={!editing} disabled={saving || editing}/>
        </FormGroup>
        <FormGroup label="Display title" labelFor="namespace-title" required>
          <InputGroup id="namespace-title" value={draft.title} onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))} placeholder="Inventory" autoFocus={editing} disabled={saving}/>
        </FormGroup>
        <FormGroup label="Description" labelFor="namespace-description">
          <TextArea id="namespace-description" value={draft.description} onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} fill rows={3} disabled={saving}/>
        </FormGroup>
        {editing && <FormGroup label="Lifecycle" labelFor="namespace-status" helperText="Archived namespaces remain attached to existing components but cannot receive new ones.">
          <HTMLSelect id="namespace-status" value={draft.status} onChange={(event) => setDraft((current) => ({ ...current, status: event.target.value }))} fill disabled={saving}>
            <option value="active">Active</option>
            <option value="archived">Archived</option>
          </HTMLSelect>
        </FormGroup>}
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" icon="floppy-disk" loading={saving}>{editing ? 'Save namespace' : 'Create namespace'}</Button></>}/>
    </form>
  </Dialog>;
}
