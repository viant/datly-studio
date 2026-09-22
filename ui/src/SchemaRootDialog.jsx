import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, InputGroup } from '@blueprintjs/core';

export function SchemaRootDialog({ api, isOpen, connector, table, sql, onClose, onCreated }) {
  const [draft, setDraft] = useState({ title: '', slug: '', viewName: '' });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    if (!isOpen) return;
    const base = table?.name ?? 'reader';
    const title = titleCase(base);
    setDraft({ title, slug: slugify(base), viewName: identifier(base) });
    setSaving(false); setError('');
  }, [isOpen, table?.name]);
  const update = (name) => (event) => setDraft((current) => ({ ...current, [name]: event.target.value }));
  const submit = async (event) => {
    event.preventDefault();
    if (!draft.title.trim() || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(draft.slug) || !/^[a-z][a-z0-9_]*$/.test(draft.viewName) || !sql.trim()) {
      setError('Enter a title, lowercase slug, view identifier, and source SQL.');
      return;
    }
    setSaving(true); setError('');
    let report;
    try {
      report = await api.createReport({ title: draft.title.trim(), slug: draft.slug, description: `Reader rooted at ${table?.schema ? `${table.schema}.` : ''}${table?.name}.`, defaultConnectorName: connector });
      const version = await api.createVersion(report.id, { authoringMode: 'dql', notes: `Root view created from ${table?.name}.` });
      const result = await api.applyReaderCommand(report.id, version.versionNo, {
        expectedSourceRevision: version.sourceRevision,
        operation: { type: 'createReader', reader: { route: `/v1/studio/readers/${draft.slug}`, name: draft.viewName, sql: sql.trim() } },
      });
      if (!result?.applied) throw new Error(result?.inspection?.diagnostics?.[0]?.message || 'Datly rejected the initial reader graph.');
      await onCreated(report);
      onClose();
    } catch (cause) {
      setError(report ? `Reader ${report.title} was created, but its initial graph failed: ${cause.message}` : cause.message);
    } finally { setSaving(false); }
  };
  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title="Create component from table" icon="new-object" canOutsideClickClose={!saving}>
    <form onSubmit={submit}><DialogBody>
      <p className="studio-dialog-lead">Create a versioned component whose reader graph is compiled by Datly from the selected SQL.</p>
      {error && <Callout intent="danger" role="alert">{error}</Callout>}
      <FormGroup label="Component title" labelFor="schema-root-title" required><InputGroup id="schema-root-title" value={draft.title} onChange={update('title')} disabled={saving} autoFocus/></FormGroup>
      <FormGroup label="Slug" labelFor="schema-root-slug" required><InputGroup id="schema-root-slug" value={draft.slug} onChange={update('slug')} disabled={saving}/></FormGroup>
      <FormGroup label="Root view identifier" labelFor="schema-root-view" helperText="Lowercase DQL alias; Go type naming is derived by Datly with Tagly." required><InputGroup id="schema-root-view" value={draft.viewName} onChange={update('viewName')} disabled={saving}/></FormGroup>
    </DialogBody><DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving}>Create component</Button></>}/></form>
  </Dialog>;
}

function words(value) { return String(value ?? '').trim().replace(/([a-z0-9])([A-Z])/g, '$1 $2').split(/[^A-Za-z0-9]+/).filter(Boolean); }
function titleCase(value) { return words(value).map((word) => word.length <= 3 ? word.toUpperCase() : word[0].toUpperCase() + word.slice(1).toLowerCase()).join(' '); }
function slugify(value) { return words(value).map((word) => word.toLowerCase()).join('-'); }
function identifier(value) { const result=words(value).map((word)=>word.toLowerCase()).join('_').replace(/^[^a-z]+/,''); return result || 'view'; }
