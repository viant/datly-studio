import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Spinner, TextArea } from '@blueprintjs/core';

const emptyDraft = () => ({ title: '', namespace: '', slug: '', description: '', defaultConnectorName: '' });

export function ReportDialog({ api, isOpen, onClose, onCreated }) {
  const [draft, setDraft] = useState(emptyDraft);
  const [connectors, setConnectors] = useState([]);
  const [namespaces, setNamespaces] = useState([]);
  const [loadingConnectors, setLoadingConnectors] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [slugTouched, setSlugTouched] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    let cancelled = false;
    setDraft(emptyDraft()); setError(''); setSlugTouched(false); setLoadingConnectors(true);
    Promise.all([api.listConnectors({ status: 'active', limit: 100 }), api.listNamespaces({ status: 'active', limit: 100 })]).then(([connectorPage, namespacePage]) => {
      if (cancelled) return;
      setConnectors(connectorPage?.items ?? []);
      const namespaceItems = namespacePage?.items ?? [];
      setNamespaces(namespaceItems);
      setDraft((current) => ({ ...current, namespace: namespaceItems.find((item) => item.name === 'general')?.name ?? namespaceItems[0]?.name ?? '' }));
    }).catch((cause) => !cancelled && setError(cause.message))
      .finally(() => !cancelled && setLoadingConnectors(false));
    return () => { cancelled = true; };
  }, [api, isOpen]);

  const onTitleChange = (event) => {
    const title = event.target.value;
    setDraft((current) => ({ ...current, title, slug: slugTouched ? current.slug : slugify(title) }));
  };
  const submit = async (event) => {
    event.preventDefault();
    if (!draft.title.trim() || !/^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$/.test(draft.namespace) || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(draft.slug) || !draft.defaultConnectorName) {
      setError('Enter a title, a dot-separated lowercase namespace, a lowercase dash-separated slug, and an active connector.');
      return;
    }
    setSaving(true); setError('');
    try {
      const report = await api.createReport({ title: draft.title.trim(), namespace: draft.namespace, slug: draft.slug, description: draft.description.trim(), defaultConnectorName: draft.defaultConnectorName });
      await onCreated(report);
      onClose();
    } catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };

  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title="New component" icon="chart" canOutsideClickClose={!saving}>
    <form onSubmit={submit}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">Create the versioned home for a dynamic, read-only Datly reader. Component identity and ownership are derived server-side.</p>
        {error && <Callout intent="danger" role="alert" style={{ marginBottom: 14 }}>{error}</Callout>}
        <FormGroup label="Component title" labelFor="report-title" required>
          <InputGroup id="report-title" value={draft.title} onChange={onTitleChange} placeholder="Vendor Catalog" autoFocus disabled={saving} />
        </FormGroup>
        <FormGroup label="Namespace" labelFor="report-namespace" helperText="Governed catalog identity. Create namespaces from the Namespaces workspace." required>
          <HTMLSelect id="report-namespace" value={draft.namespace} onChange={(event) => setDraft((current) => ({ ...current, namespace: event.target.value }))} fill disabled={saving || namespaces.length === 0}>
            <option value="">Select a namespace</option>
            {namespaces.map((item) => <option value={item.name} key={`${item.ownerId}:${item.name}`}>{item.title} · {item.name}</option>)}
          </HTMLSelect>
          {!loadingConnectors && namespaces.length === 0 && <Callout intent="warning" compact style={{ marginTop: 8 }}>Create an active namespace before creating a component.</Callout>}
        </FormGroup>
        <FormGroup label="Slug" labelFor="report-slug" helperText="Lowercase letters, numbers, and dashes; used in reader identity." required>
          <InputGroup id="report-slug" value={draft.slug} onChange={(event) => { setSlugTouched(true); setDraft((current) => ({ ...current, slug: event.target.value })); }} placeholder="vendor-catalog" disabled={saving} />
        </FormGroup>
        <FormGroup label="Active connector" labelFor="report-connector" required helperText="Only tested active connectors can back a reader.">
          {loadingConnectors ? <Spinner size={20} /> : <HTMLSelect id="report-connector" value={draft.defaultConnectorName} onChange={(event) => setDraft((current) => ({ ...current, defaultConnectorName: event.target.value }))} fill disabled={saving || connectors.length === 0}>
            <option value="">Select a connector</option>
            {connectors.map((connector) => <option value={connector.name} key={connector.name}>{connector.name} · {connector.driver}</option>)}
          </HTMLSelect>}
        </FormGroup>
        <FormGroup label="Description" labelFor="report-description">
          <TextArea id="report-description" value={draft.description} onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} fill rows={2} disabled={saving} />
        </FormGroup>
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving} disabled={loadingConnectors || connectors.length === 0 || namespaces.length === 0}>Create component</Button></>} />
    </form>
  </Dialog>;
}

function slugify(value) {
  return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
}
