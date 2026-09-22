import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';

export function ReaderSubviewDialog({ isOpen, structure, initialParent = '', onClose, onApply }) {
  const views = structure?.views ?? [];
  const [draft, setDraft] = useState({ name: '', kind: 'subview', parent: '', sql: '', on: '' });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const parentDefault = useMemo(() => views[0]?.name ?? '', [views]);

  useEffect(() => { if (isOpen) { setDraft({ name: '', kind: 'subview', parent: initialParent || parentDefault, sql: '', on: '' }); setSaving(false); setError(''); } }, [isOpen, parentDefault, initialParent]);
  const update = (name) => (event) => setDraft((current) => ({ ...current, [name]: event.target.value }));
  const submit = async (event) => {
    event.preventDefault();
    const joined = draft.kind === 'subview';
    if (!/^[A-Za-z][A-Za-z0-9_]*$/.test(draft.name) || !draft.parent || !draft.sql.trim() || (joined && !draft.on.trim())) {
      setError(joined ? 'Enter an identifier, parent view, source SQL, and complete relation key expression.' : 'Enter an identifier, root view, and derived SQL.');
      return;
    }
    setSaving(true); setError('');
    try {
      await onApply({ type: 'addView', view: { name: draft.name, kind: draft.kind, parent: joined ? draft.parent : parentDefault, join: joined ? 'JOIN' : '', sql: draft.sql.trim(), on: joined ? draft.on.trim() : '' } });
      onClose();
    } catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };

  const joined = draft.kind === 'subview';
  return <Dialog className="studio-connector-dialog studio-subview-dialog" isOpen={isOpen} onClose={onClose} title="Add view" icon="git-branch" canOutsideClickClose={!saving}>
    <form onSubmit={submit}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">Create a regular joined subview or a query-derived output. Datly compiles the typed relation before the version is updated.</p>
        {error && <Callout intent="danger" role="alert" style={{ marginBottom: 14 }}>{error}</Callout>}
        <div className="studio-form-grid">
          <FormGroup label="View name" labelFor="subview-name" helperText="Stable DQL view identifier." required><InputGroup id="subview-name" value={draft.name} onChange={update('name')} placeholder={joined?'contacts':'totals'} autoFocus disabled={saving} /></FormGroup>
          <FormGroup label="View type" labelFor="subview-kind" required><HTMLSelect id="subview-kind" value={draft.kind} onChange={update('kind')} fill disabled={saving}><option value="subview">Regular join</option><option value="derived">Derived view</option></HTMLSelect></FormGroup>
        </div>
        <FormGroup label={joined?'Parent view':'Root view'} labelFor="subview-parent" required><HTMLSelect id="subview-parent" value={joined?draft.parent:parentDefault} onChange={update('parent')} fill disabled={saving||!joined}>{views.map((view,index) => <option key={view.name} value={view.name} disabled={!joined&&index!==0}>{view.name}</option>)}</HTMLSelect></FormGroup>
        <FormGroup label={joined?'Subview SQL':'Derived SQL'} labelFor="subview-sql" helperText={joined?'One inner-view query; test it before applying the relation.':'A typed output query such as totals or bounds; it is not joined into each root row.'} required><div id="subview-sql" className="studio-code-editor"><Editor ariaLabel={joined?'Subview SQL source':'Derived SQL source'} value={draft.sql} onChange={(value)=>setDraft((current)=>({...current,sql:value}))} language="sql" height="180px" readOnly={saving}/></div></FormGroup>
        {joined && <FormGroup label="Relation keys" labelFor="subview-on" helperText="Use complete parent/child keys, for example contacts.VENDOR_ID=vendor.ID." required><InputGroup id="subview-on" value={draft.on} onChange={update('on')} placeholder="contacts.VENDOR_ID=vendor.ID" disabled={saving} /></FormGroup>}
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button type="submit" intent="primary" loading={saving}>Add {joined?'joined view':'derived view'}</Button></>} />
    </form>
  </Dialog>;
}
