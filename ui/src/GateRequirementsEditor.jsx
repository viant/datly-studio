import React, {useEffect, useRef, useState} from 'react';
import {Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, Spinner, Tag} from '@blueprintjs/core';
import './resourceAccess.css';

const clone = value => JSON.parse(JSON.stringify(value));
const options = (items = [], selected = []) => {
  const values = items.map(item => typeof item === 'string' ? {id: item, label: item} : item);
  for (const id of selected) if (id && !values.some(item => item.id === id)) values.unshift({id, label: id});
  return values;
};

// Host APIs own identity, management authority, provider choices and CAS.
// The editor only constructs a versioned requirements draft for review.
export function GateRequirementsEditor({api, resource, action, previewSelection = [], readOnly = false, onStateChange}) {
  const [document, setDocument] = useState(null);
  const [draft, setDraft] = useState(null);
  const [context, setContext] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [review, setReview] = useState(false);
  const [error, setError] = useState(null);
  const [saved, setSaved] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [previewDecision, setPreviewDecision] = useState(null);
  const [previewError, setPreviewError] = useState(false);
  const [previewCheckedAt, setPreviewCheckedAt] = useState('');
  const sequence = useRef(0);
  const previewSequence = useRef(0);
  const identity = JSON.stringify([resource, action]);
  const load = async () => {
    const ticket = ++sequence.current;
    previewSequence.current++;
    setPreviewDecision(null); setPreviewError(false); setPreviewing(false); setPreviewCheckedAt('');
    setLoading(true); setError(null); setSaved(false); setReview(false);
    try {
      const [value, editorContext] = await Promise.all([
        api.getGateRequirements(resource, action),
        api.getGateRequirementsContext ? api.getGateRequirementsContext(resource, action) : Promise.resolve(null),
      ]);
      if (ticket !== sequence.current) return;
      if (!value?.revision || value?.requirements?.schemaVersion !== 1) throw new Error('Unsupported gate requirements document');
      setDocument(value); setDraft(clone(value.requirements)); setContext(editorContext);
    } catch (cause) { if (ticket === sequence.current) setError(cause); }
    finally { if (ticket === sequence.current) setLoading(false); }
  };
  useEffect(() => {
    setDocument(null); setDraft(null); setContext(null); setSaving(false); load();
    return () => { sequence.current++; previewSequence.current++; };
  }, [api, identity]);
  const dirty = Boolean(document && draft && JSON.stringify(document.requirements) !== JSON.stringify(draft));
  const validDraft = !draft?.entity || Boolean(draft.entity.permission && draft.entity.selectionParameter);
  const validEntitlement = !draft?.entitlement || Boolean(draft.entitlement.key && (draft.entitlement.scope !== 'entity' || draft.entity));
  useEffect(() => { onStateChange?.({dirty, saving}); }, [dirty, saving, onStateChange]);
  const disabled = loading || saving || readOnly || context?.canManage !== true;
  const choices = context?.choices || {};
  const update = patch => { setDraft(current => ({...current, ...patch})); setSaved(false); };
  const toggle = (field, id) => {
    const current = draft[field] || [];
    update({[field]: current.includes(id) ? current.filter(item => item !== id) : [...current, id]});
  };
  const save = async () => {
    if (!review || !dirty || disabled || !validDraft || !validEntitlement) return;
    previewSequence.current++; setPreviewDecision(null); setPreviewError(false); setPreviewing(false); setPreviewCheckedAt('');
    const ticket = sequence.current;
    setSaving(true); setError(null);
    try {
      const updated = await api.replaceGateRequirements({resource, action, expectedRevision: document.revision, requirements: clone(draft)});
      if (ticket !== sequence.current) return;
      if (!updated?.revision || updated.revision === document.revision) throw new Error('Gate revision did not advance');
      setDocument(updated); setDraft(clone(updated.requirements)); setSaved(true); setReview(false);
      previewSequence.current++; setPreviewDecision(null); setPreviewError(false); setPreviewing(false); setPreviewCheckedAt('');
    } catch (cause) { if (ticket === sequence.current) { setError(cause); setReview(false); } }
    finally { if (ticket === sequence.current) setSaving(false); }
  };
  const checkCurrentAccess = async () => {
    if (!document || typeof api.checkCurrentGate !== 'function') return;
    const ticket = ++previewSequence.current;
    const loadTicket = sequence.current;
    const revision = document.revision;
    setPreviewing(true); setPreviewError(false); setPreviewDecision(null); setPreviewCheckedAt('');
    try {
      const decision = await api.checkCurrentGate(resource, action, previewSelection);
      if (ticket !== previewSequence.current || loadTicket !== sequence.current) return;
      if (decision?.requirementsRevision !== revision) throw new Error('Gate revision changed');
      setPreviewDecision(decision); setPreviewCheckedAt(new Date().toLocaleTimeString());
    } catch (_) { if (ticket === previewSequence.current && loadTicket === sequence.current) setPreviewError(true); }
    finally { if (ticket === previewSequence.current && loadTicket === sequence.current) setPreviewing(false); }
  };
  useEffect(() => {
    if (!previewDecision) return undefined;
    const lease = Date.parse(previewDecision.validUntil) - Date.now();
    if (!Number.isFinite(lease) || lease <= 0) { setPreviewDecision(null); return undefined; }
    const timer = setTimeout(() => setPreviewDecision(null), Math.min(lease, 2147483647));
    return () => clearTimeout(timer);
  }, [previewDecision]);
  const conflict = error?.code === 'conflict';
  return <section className="studio-resource-access" aria-label="Gate requirements">
    <header className="studio-resource-access-heading"><div><h2>Additional access requirements</h2><p>{resource.kind} · <strong>{resource.id}</strong> · {action}</p></div>{document && <Tag minimal>Gate revision {document.revision}</Tag>}</header>
    <p>These requirements apply alongside the resource policy. An account feature, role, entity permission, or entitlement cannot override an ACL denial.</p>
    {error && <Callout intent={conflict ? 'warning' : 'danger'} role="alert" title={conflict ? 'Requirements changed elsewhere' : 'Requirements unavailable'}>{conflict ? 'Your draft is preserved. Reload the current revision before saving.' : error.message}<Button minimal disabled={saving} onClick={load}>{dirty ? 'Discard draft and reload' : 'Retry'}</Button></Callout>}
    {loading && <div role="status"><Spinner size={22}/> Loading requirements…</div>}
    {!loading && draft && <div className="studio-resource-access-detail">
      <fieldset disabled={disabled}><legend>Required account features</legend>{options(choices.exposures, draft.requiredExposures).map(item => <label key={item.id}><input type="checkbox" checked={(draft.requiredExposures || []).includes(item.id)} onChange={() => toggle('requiredExposures', item.id)}/> {item.label || item.id}</label>)}</fieldset>
      <fieldset disabled={disabled}><legend>Allowed user roles</legend><p>Any selected role matches. Every required feature above still applies.</p>{options(choices.roles, draft.allowedRoles).map(item => <label key={item.id}><input type="checkbox" checked={(draft.allowedRoles || []).includes(item.id)} onChange={() => toggle('allowedRoles', item.id)}/> {item.label || item.id}</label>)}</fieldset>
      <FormGroup label="Entity requirement" labelFor="gate-entity-type"><HTMLSelect id="gate-entity-type" disabled={disabled} value={draft.entity?.type || ''} onChange={event => update({entity: event.target.value ? {type: event.target.value, permission: '', selectionParameter: '', selectionMode: 'single'} : undefined})}><option value="">No entity requirement</option>{options(choices.entityTypes, draft.entity?.type ? [draft.entity.type] : []).map(item => <option key={item.id} value={item.id}>{item.label || item.id}</option>)}</HTMLSelect></FormGroup>
      {draft.entity && <>
        <FormGroup label="Entity permission" labelFor="gate-entity-permission"><HTMLSelect id="gate-entity-permission" disabled={disabled} value={draft.entity.permission} onChange={event => update({entity: {...draft.entity, permission: event.target.value}})}><option value="">Choose permission</option>{options(choices.entityPermissions?.[draft.entity.type], draft.entity.permission ? [draft.entity.permission] : []).map(item => <option key={item.id} value={item.id}>{item.label || item.id}</option>)}</HTMLSelect></FormGroup>
        <FormGroup label="Selection parameter" labelFor="gate-selection-parameter"><HTMLSelect id="gate-selection-parameter" disabled={disabled} value={draft.entity.selectionParameter || ''} onChange={event => update({entity: {...draft.entity, selectionParameter: event.target.value}})}><option value="">Choose trusted parameter</option>{options(choices.selectionParameters, draft.entity.selectionParameter ? [draft.entity.selectionParameter] : []).map(item => <option key={item.id} value={item.id}>{item.label || item.id}</option>)}</HTMLSelect></FormGroup>
        <FormGroup label="Selection mode" labelFor="gate-selection-mode"><HTMLSelect id="gate-selection-mode" disabled={disabled} value={draft.entity.selectionMode} onChange={event => update({entity: {...draft.entity, selectionMode: event.target.value}})}><option value="single">One entity</option><option value="multiple">All selected entities</option></HTMLSelect></FormGroup>
      </>}
      <FormGroup label="Entitlement provider" labelFor="gate-entitlement-provider"><HTMLSelect id="gate-entitlement-provider" disabled={disabled} value={draft.entitlement?.providerRef || ''} onChange={event => update({entitlement: event.target.value ? {providerRef: event.target.value, key: '', scope: 'account'} : undefined})}><option value="">No entitlement requirement</option>{options(choices.entitlementProviders, draft.entitlement?.providerRef ? [draft.entitlement.providerRef] : []).map(item => <option key={item.id} value={item.id}>{item.label || item.id}</option>)}</HTMLSelect></FormGroup>
      {draft.entitlement && <>
        <FormGroup label="Entitlement key" labelFor="gate-entitlement-key"><HTMLSelect id="gate-entitlement-key" disabled={disabled} value={draft.entitlement.key} onChange={event => update({entitlement: {...draft.entitlement, key: event.target.value}})}><option value="">Choose entitlement</option>{options(choices.entitlementKeys?.[draft.entitlement.providerRef], draft.entitlement.key ? [draft.entitlement.key] : []).map(item => <option key={item.id} value={item.id}>{item.label || item.id}</option>)}</HTMLSelect></FormGroup>
        <FormGroup label="Entitlement scope" labelFor="gate-entitlement-scope"><HTMLSelect id="gate-entitlement-scope" disabled={disabled} value={draft.entitlement.scope} onChange={event => update({entitlement: {...draft.entitlement, scope: event.target.value}})}><option value="account">Account</option><option value="entity" disabled={!draft.entity}>Selected entity</option></HTMLSelect></FormGroup>
      </>}
    </div>}
    {document && <footer className="studio-resource-access-footer"><span role="status">{disabled ? 'You have read-only access.' : saved ? 'Requirements saved.' : dirty ? 'Unsaved requirement changes' : 'All changes saved'}</span><div>{typeof api.checkCurrentGate === 'function' && <Button minimal loading={previewing} disabled={loading || saving} onClick={checkCurrentAccess}>Check my current access</Button>}{!disabled && <Button intent="primary" disabled={!dirty || conflict || !validDraft || !validEntitlement} onClick={() => setReview(true)}>Review requirement changes</Button>}</div></footer>}
    {document && typeof api.checkCurrentGate === 'function' && <p>This check uses the saved revision and your current verified identity{dirty ? ', not your unsaved draft' : ''}.</p>}
    {previewDecision && <Callout role="status" intent={previewDecision.effect === 'allow' ? 'success' : 'warning'} title={previewDecision.effect === 'allow' ? 'Allowed at last check' : 'Denied at last check'}>{previewDecision.effect === 'allow' ? 'Your account passed the saved policy and requirements.' : `The saved policy and requirements denied this check${previewDecision.reasonCode ? ` (${previewDecision.reasonCode})` : ''}.`} Checked at {previewCheckedAt}.</Callout>}
    {previewError && <Callout role="alert" intent="danger">Current access could not be checked. Reload and try again.</Callout>}
    <Dialog isOpen={review && dirty} title="Review requirement changes" onClose={() => !saving && setReview(false)}><DialogBody><p>Saving asks the server to compare and replace gate revision {document?.revision}. This review does not compute effective access.</p><dl><dt>Current</dt><dd><pre>{JSON.stringify(document?.requirements, null, 2)}</pre></dd><dt>Proposed</dt><dd><pre>{JSON.stringify(draft, null, 2)}</pre></dd></dl></DialogBody><DialogFooter actions={<><Button disabled={saving} onClick={() => setReview(false)}>Back to editor</Button><Button intent="primary" loading={saving} disabled={disabled} onClick={save}>Save requirements</Button></>}/></Dialog>
  </section>;
}
