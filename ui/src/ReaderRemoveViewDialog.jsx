import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Tag } from '@blueprintjs/core';

// ReaderRemoveViewDialog is deliberately a confirmation boundary. It only
// creates the typed Datly reader-builder operation; DQL is never changed in
// the browser.
export function ReaderRemoveViewDialog({ isOpen, structure, initialView = '', onClose, onApply }) {
  const candidates = useMemo(() => removableViews(structure), [structure]);
  const [view, setView] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const selected = candidates.find((item) => item.name === view);

  useEffect(() => {
    if (!isOpen) return;
    setView(candidates.some((item)=>item.name===initialView) ? initialView : candidates[0]?.name ?? '');
    setConfirmation(''); setSaving(false); setError('');
  }, [isOpen, candidates, initialView]);

  const submit = async (event) => {
    event.preventDefault();
    if (!selected) { setError('There is no removable related view in this reader.'); return; }
    if (selected.children.length) { setError(`Move or remove ${selected.children.join(', ')} before removing ${selected.name}.`); return; }
    if (confirmation !== selected.name) { setError(`Type ${selected.name} exactly to confirm this source change.`); return; }
    setSaving(true); setError('');
    try {
      await onApply({ type: 'removeView', view: { name: selected.name } });
      onClose();
    } catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };

  return <Dialog className="studio-connector-dialog" isOpen={isOpen} onClose={onClose} title="Remove related view" icon="trash" canOutsideClickClose={!saving}>
    <form onSubmit={submit}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">This creates a new immutable source revision. Datly removes the selected related view and its outer projection; it does not run ad-hoc SQL in the browser.</p>
        {error && <Callout intent="danger" role="alert" style={{ marginBottom: 14 }}>{error}</Callout>}
        {candidates.length === 0 ? <Callout intent="warning">The root view cannot be removed. Add a related view before using this operation.</Callout> : <>
          <FormGroup label="Related view" labelFor="remove-view"><HTMLSelect id="remove-view" value={view} onChange={(event) => { setView(event.target.value); setConfirmation(''); setError(''); }} fill disabled={saving}>{candidates.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</HTMLSelect></FormGroup>
          {selected && <div className="studio-impact-panel" aria-live="polite"><strong>Impact</strong><div><Tag intent="danger" minimal>Remove</Tag> <code>{selected.name}</code> source and its output projection.</div>{selected.children.length > 0 ? <Callout intent="warning" compact>This view has dependent subviews: {selected.children.join(', ')}. Removal is blocked until they are moved or removed.</Callout> : <div><Tag intent="success" minimal>Safe shape</Tag> No dependent subviews were found in Datly’s compiled graph.</div>}</div>}
          <FormGroup label={`Type ${selected?.name ?? 'the view name'} to confirm`} labelFor="remove-confirmation" helperText="This prevents an accidental destructive source edit."><InputGroup id="remove-confirmation" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} autoComplete="off" disabled={saving} /></FormGroup>
        </>}
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="danger" icon="trash" loading={saving} disabled={!selected || selected.children.length > 0 || confirmation !== selected.name}>Remove view</Button></>} />
    </form>
  </Dialog>;
}

function removableViews(structure) {
  const root = structure?.component?.rootView;
  const sourceViews = new Set((structure?.views ?? []).map((view) => view.name));
  if (!root) return [];
  const result = [];
  const visit = (view) => {
    for (const relation of view?.relations ?? []) {
      const child = relation.view;
      const name = child?.namespace || relation.name;
      const descendants = flattenChildren(child).map((item) => item.namespace || item.name).filter(Boolean);
      if (sourceViews.has(name)) result.push({ name, children: descendants });
      visit(child);
    }
  };
  visit(root);
  return result;
}

function flattenChildren(view) {
  const result = [];
  for (const relation of view?.relations ?? []) {
    if (relation.view) { result.push(relation.view); result.push(...flattenChildren(relation.view)); }
  }
  return result;
}
