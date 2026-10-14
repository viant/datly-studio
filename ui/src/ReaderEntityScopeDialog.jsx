import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup } from '@blueprintjs/core';

function viewsOf(view) {
  return view ? [view, ...(view.relations || []).flatMap(relation => viewsOf(relation.view))] : [];
}

export function ReaderEntityScopeDialog({ isOpen, report, root, structure, onClose, onApply }) {
  const views = viewsOf(root);
  const [draft, setDraft] = useState({ entity: 'customer', view: '', qualifier: '', column: 'id', idType: 'number' });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    if (!isOpen) return;
    setDraft({ entity: 'customer', view: root?.namespace || root?.name || '', qualifier: String(root?.source?.table || '').split('.').at(-1) || '', column: 'id', idType: 'number' });
    setError(''); setSaving(false);
  }, [isOpen, report?.id]);
  const change = key => event => setDraft(value => ({ ...value, [key]: event.target.value }));
  const submit = async event => {
    event.preventDefault();
    const identifier = /^[A-Za-z_][A-Za-z0-9_]*$/;
    if (!identifier.test(draft.entity) || !identifier.test(draft.column) || (draft.qualifier && !identifier.test(draft.qualifier)) || !views.some(view => (view.namespace || view.name) === draft.view)) {
      setError('Choose a view and enter valid entity, column and SQL qualifier names.'); return;
    }
    if ((structure?.declarations || []).some(({parameter}) => ['Auth', 'AuthorizedEntityIDs'].includes(parameter?.name))) {
      setError('This component already defines Auth or AuthorizedEntityIDs. Review its existing scope binding first.'); return;
    }
    setSaving(true); setError('');
    try {
      await onApply({ type: 'batch', operations: [
        { type: 'addField', field: { name: 'Auth', type: '*github.com/viant/datly-studio/runtime/accesscontext.Output', sourceKind: 'component', sourceName: `GET:/_studio/access/context/${report.id}/${draft.entity}`, required: true } },
        { type: 'addField', field: { name: 'AuthorizedEntityIDs', type: draft.idType === 'number' ? '[]string,[]int' : '[]string', sourceKind: 'param', sourceName: 'Auth.Scope.IDs', required: true, ...(draft.idType === 'number' ? {codec: {name: 'EntityIDs'}} : {}) } },
        { type: 'addFieldPredicate', predicate: { field: 'AuthorizedEntityIDs', view: draft.view, group: 99, name: 'in', args: [draft.qualifier, draft.column], groupOperator: 'AND', applyWhenAbsent: true } },
      ] });
      onClose();
    } catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };
  return <Dialog className="studio-connector-dialog" isOpen={isOpen} title="Bind row access scope" onClose={() => !saving && onClose()} canOutsideClickClose={!saving} canEscapeKeyClose={!saving}>
    <form onSubmit={submit}><DialogBody>
      <p>Filter this view to entity IDs supplied by the verified access policy. Callers cannot supply or replace these IDs.</p>
      {error && <Callout intent="danger" role="alert">{error}</Callout>}
      <FormGroup label="Entity type" labelFor="scope-entity"><InputGroup id="scope-entity" value={draft.entity} onChange={change('entity')} disabled={saving}/></FormGroup>
      <FormGroup label="Scoped view" labelFor="scope-view"><HTMLSelect id="scope-view" fill value={draft.view} onChange={change('view')} disabled={saving}>{views.map(view => <option key={view.namespace || view.name} value={view.namespace || view.name}>{view.namespace || view.name}</option>)}</HTMLSelect></FormGroup>
      <div className="studio-form-grid">
        <FormGroup label="Entity ID column" labelFor="scope-column"><InputGroup id="scope-column" value={draft.column} onChange={change('column')} disabled={saving}/></FormGroup>
        <FormGroup label="SQL qualifier" labelFor="scope-qualifier" helperText="The table name or alias inside this view's query; leave empty for an unqualified column."><InputGroup id="scope-qualifier" value={draft.qualifier} onChange={change('qualifier')} disabled={saving}/></FormGroup>
      </div>
      <FormGroup label="Entity ID type" labelFor="scope-type"><HTMLSelect id="scope-type" fill value={draft.idType} onChange={change('idType')} disabled={saving}><option value="number">Integer</option><option value="string">Text</option></HTMLSelect></FormGroup>
      <Callout compact>Set the matching required entity scope in the component's consumption policy. The runtime denies execution when the policy cannot supply that scope.</Callout>
    </DialogBody><DialogFooter actions={<><Button disabled={saving} onClick={onClose}>Cancel</Button><Button type="submit" intent="primary" loading={saving}>Bind entity scope</Button></>}/></form>
  </Dialog>;
}
