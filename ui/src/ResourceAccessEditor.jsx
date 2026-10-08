import React, { useEffect, useRef, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Spinner, Tag } from '@blueprintjs/core';
import { changedPolicies, describePolicy } from './resourceAccessReview.js';
import './resourceAccess.css';

const publicActions = new Set(['discover', 'describe', 'execute', 'retrieve']);
const blankRule = () => ({ kind: 'role', value: '' });
const clone = value => JSON.parse(JSON.stringify(value));

// Resource types and selector catalogs belong to the embedding application.
// This editor never interprets JWTs or computes an authoritative access decision.
export function ResourceAccessEditor({ api, resource, actions, choices = {}, readOnly = false, displayName, onStateChange }) {
  const [document, setDocument] = useState(null);
  const [draft, setDraft] = useState(null);
  const [action, setAction] = useState(actions[0]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [saved, setSaved] = useState(false);
  const [editorContext, setEditorContext] = useState(null);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [preview, setPreview] = useState(null);
  const [previewing, setPreviewing] = useState(false);
  const [previewError, setPreviewError] = useState(false);
  const sequence = useRef(0);
  const previewSequence = useRef(0);
  const resourceKey = JSON.stringify(resource);
  const clearPreview = () => { previewSequence.current++; setPreview(null); setPreviewing(false); setPreviewError(false); };
  const load = async () => {
    const ticket = ++sequence.current;
    clearPreview();
    setLoading(true); setError(null); setSaved(false); setReviewOpen(false);
    try {
      const [value, context] = await Promise.all([
        api.getResourceAccess(resource),
        api.getResourceAccessContext ? api.getResourceAccessContext(resource) : Promise.resolve(null),
      ]);
      if (ticket !== sequence.current) return;
      setDocument(value); setDraft(clone(value)); setEditorContext(context);
      if (readOnly) setAction(actions.find(name => value?.policies?.[name]) || actions[0]);
    } catch (cause) { if (ticket === sequence.current) setError(cause); }
    finally { if (ticket === sequence.current) setLoading(false); }
  };
  useEffect(() => {
    setDocument(null); setDraft(null); setAction(actions[0]); setSaving(false); setReviewOpen(false); load();
    return () => { sequence.current++; previewSequence.current++; };
  }, [api, resourceKey]);
  useEffect(() => { clearPreview(); }, [action]);
  const update = policy => {
    setDraft(current => ({ ...current, policies: { ...current.policies, [action]: policy } }));
    setSaved(false); clearPreview();
  };
  const changes = changedPolicies(document, draft);
  const dirty = changes.length > 0;
  useEffect(()=>{onStateChange?.({dirty,saving});},[dirty,saving,onStateChange]);
  const save = async () => {
    if (incompleteRules || !reviewOpen || !dirty || readOnly || editorContext?.canManage === false) return;
    const ticket = sequence.current;
    setSaving(true); setError(null);
    try {
      const result = await api.replaceResourceAccess(draft);
      if (ticket !== sequence.current) return;
      setDocument(result); setDraft(clone(result)); setSaved(true); setReviewOpen(false); clearPreview();
    } catch (cause) { if (ticket === sequence.current) { setError(cause); setReviewOpen(false); } }
    finally { if (ticket === sequence.current) setSaving(false); }
  };
  const policy = draft?.policies?.[action];
  const checkCurrentAccess = async () => {
    if (!document || typeof api.checkCurrentAccess !== 'function') return;
    const ticket = ++previewSequence.current;
    const loadTicket = sequence.current;
    setPreview(null); setPreviewError(false); setPreviewing(true);
    try {
      const decision = await api.checkCurrentAccess(resource, action);
      if (ticket !== previewSequence.current || loadTicket !== sequence.current) return;
      if (!decision || !['allow', 'deny'].includes(decision.effect)) throw new Error('Authorization check is invalid');
      setPreview({decision, checkedAt: new Date().toLocaleTimeString()});
    } catch (_) { if (ticket === previewSequence.current && loadTicket === sequence.current) setPreviewError(true); }
    finally { if (ticket === previewSequence.current && loadTicket === sequence.current) setPreviewing(false); }
  };
  useEffect(() => {
    if (!preview) return undefined;
    const timer = setTimeout(() => setPreview(null), 10_000);
    return () => clearTimeout(timer);
  }, [preview]);
  const cannotManage = readOnly || editorContext?.canManage === false;
  const disabled = cannotManage || saving || loading;
  const incompleteRules = Object.values(draft?.policies ?? {}).some(policy => policy.mode === 'protected' && policy.rule && !completeRule(policy.rule));
  const providerChoices = editorContext?.choices || choices;
  const conflict = error?.code === 'conflict';
  return <section className="studio-resource-access" aria-label="Resource access">
    <header className="studio-resource-access-heading">
      <div><h2>{displayName ? `${displayName} permissions` : 'Access permissions'}</h2><p>{resource.kind} · <strong>{resource.id}</strong> · {resource.tenant} · version {resource.version}</p></div>
      {document && <Tag minimal>Policy revision {document.revision}</Tag>}
    </header>
    {error && <Callout intent={conflict ? 'warning' : 'danger'} role="alert" title={conflict ? 'Permissions changed elsewhere' : 'Access request failed'}>
      {conflict ? 'Your edits are preserved. Reload the saved policy to review the latest permissions before making changes again.' : error.message}
      <Button minimal onClick={load} disabled={saving}>{dirty ? 'Discard edits and reload' : 'Retry'}</Button>
    </Callout>}
    {loading && <div role="status" className="studio-resource-access-loading"><Spinner size={22}/> Loading permissions…</div>}
    {!loading && draft && <div className="studio-resource-access-layout">
      <div className="studio-resource-action-picker"><FormGroup label="Permission action" labelFor="resource-access-action"><HTMLSelect id="resource-access-action" fill value={action} disabled={saving} onChange={event => setAction(event.target.value)}>{actions.map(name => <option key={name} value={name}>{name} · {draft.policies[name]?.mode === 'public' ? 'Public' : draft.policies[name] ? 'Protected' : 'Denied'}</option>)}</HTMLSelect></FormGroup></div>
      <nav aria-label="Permission actions">{actions.map(name => <button key={name} type="button" disabled={saving} aria-current={name === action ? 'page' : undefined} onClick={() => setAction(name)}>
        <strong>{name}</strong><span>{draft.policies[name]?.mode === 'public' ? 'Public' : draft.policies[name] ? 'Protected' : 'Denied · no policy'}</span>
      </button>)}</nav>
      <div className="studio-resource-access-detail">
        <h3>{action}</h3>
        <p>Tenant boundaries and required source permissions apply independently of these rules.</p>
        {editorContext?.source === 'verified-principal' && <p>Choices come from your verified identity. An administrator can configure a provider directory for organization-wide choices.</p>}
        <FormGroup label="Access mode" labelFor="resource-access-mode">
          <HTMLSelect id="resource-access-mode" disabled={disabled} value={policy?.mode || ''} onChange={event => {
            const mode = event.target.value;
            if (!mode) { const policies = { ...draft.policies }; delete policies[action]; setDraft({ ...draft, policies }); clearPreview(); }
            else update(mode === 'public' ? { mode } : { mode, rule: blankRule() });
          }}>
            <option value="">Deny access</option><option value="protected">Protected</option>
            {publicActions.has(action) && <option value="public">Public consumption</option>}
          </HTMLSelect>
        </FormGroup>
        {policy?.mode === 'public' && <Callout icon="globe">No identity rule is required for this action. Dependencies can still restrict access.</Callout>}
        {policy?.mode === 'protected' && <>
          <RuleEditor rule={policy.rule || blankRule()} onChange={rule => update({ ...policy, rule })} choices={providerChoices} disabled={disabled}/>
          <FormGroup label="Required entity scope" labelFor="resource-access-scope" helperText="Applied to every matching rule, including any-match groups.">
            <HTMLSelect id="resource-access-scope" disabled={disabled} value={policy.entityType || ''} onChange={event => update({ ...policy, entityType: event.target.value })}>
              <option value="">No entity scope</option>
              {[...new Set([...(providerChoices.entityTypes || []), ...(policy.entityType ? [policy.entityType] : [])])].map(type => <option key={type}>{type}</option>)}
            </HTMLSelect>
          </FormGroup>
        </>}
      </div>
    </div>}
    {document && <footer className="studio-resource-access-footer">
      <span role="status">{cannotManage ? 'You have read-only access.' : saved ? 'Permissions saved.' : dirty ? 'Unsaved permission changes' : 'All changes saved'}</span>
      <span>{typeof api.checkCurrentAccess === 'function' && <Button minimal loading={previewing} disabled={loading || saving} onClick={checkCurrentAccess}>Check my current access</Button>}{!cannotManage && <Button intent="primary" icon="eye-open" disabled={disabled || !dirty || conflict || incompleteRules} onClick={() => setReviewOpen(true)}>Review changes</Button>}</span>
    </footer>}
    {document && typeof api.checkCurrentAccess === 'function' && <p>This check uses the saved policy and your current verified identity{dirty ? ', not your unsaved draft' : ''}.</p>}
    {preview && <Callout role="status" intent={preview.decision.effect === 'allow' ? 'success' : 'warning'} title={preview.decision.effect === 'allow' ? 'Allowed at last check' : 'Denied at last check'}>{preview.decision.effect === 'allow' ? preview.decision.bounded ? 'Access is limited to your currently allowed entities.' : 'The saved policy allowed this action.' : 'The saved policy denied this action.'} Checked at {preview.checkedAt}.</Callout>}
    {previewError && <Callout role="alert" intent="danger">Current access could not be checked. Reload and try again.</Callout>}
    <Dialog className="studio-resource-review-dialog" isOpen={reviewOpen && dirty} title="Review permission changes" icon="eye-open" onClose={() => !saving && setReviewOpen(false)} canEscapeKeyClose={!saving} canOutsideClickClose={!saving}>
      <DialogBody>
        <p>Review {resource.kind} <strong>{displayName || resource.id}</strong> ({resource.id}) at policy revision {document?.revision}. Saving asks the server to compare and replace this exact revision.</p>
        <div className="studio-resource-review-changes">{changes.map(change => <section key={change.action} aria-label={`${change.action} change`}>
          <h3>{change.action}</h3><dl><div><dt>Current</dt><dd>{describePolicy(change.before)}</dd></div><div><dt>Proposed</dt><dd>{describePolicy(change.after)}</dd></div></dl>
        </section>)}</div>
        <Callout intent="primary">This is a policy-configuration review, not an effective-access decision. The server rechecks your identity and current revision when saving.</Callout>
        <details className="studio-resource-review-raw"><summary>Exact proposed policy JSON</summary><pre>{JSON.stringify(draft?.policies || {}, null, 2)}</pre></details>
      </DialogBody>
      <DialogFooter actions={<><Button disabled={saving} onClick={() => setReviewOpen(false)}>Back to editor</Button><Button intent="primary" icon="floppy-disk" loading={saving} disabled={saving || conflict || cannotManage || incompleteRules} onClick={save}>Save permissions</Button></>}/>
    </Dialog>
  </section>;
}

function RuleEditor({ rule, onChange, choices, disabled, path = 'rule', depth = 0 }) {
  const group = rule.kind === 'all' || rule.kind === 'any';
  const options = choices[rule.kind] || [];
  const value = rule.kind === 'entity' ? JSON.stringify(rule.entity || {}) : rule.value || '';
  const [query, setQuery] = useState('');
  const optionsWithCurrent = [...options];
  const optionValue = item => rule.kind === 'entity' ? JSON.stringify(item.entity) : item.id;
  if (value && !options.some(item => optionValue(item) === value)) optionsWithCurrent.unshift({ id: value, entity: rule.entity, label: rule.kind === 'entity' ? `${rule.entity?.type}: ${rule.entity?.id}` : value });
  return <fieldset className="studio-resource-rule" disabled={disabled}>
    <legend>Access rule</legend>
    <HTMLSelect aria-label={`${path} condition`} value={rule.kind} onChange={event => {
      const kind = event.target.value;
      onChange(kind === 'all' || kind === 'any' ? { kind, rules: [blankRule()] } : kind === 'entity' ? { kind, entity: { type: '', id: '' } } : { kind, value: '' });
    }}>
      <option value="subject">Principal</option><option value="role">Role</option><option value="exposure">Feature exposure</option><option value="entity">Allowed entity</option>
      {depth < 12 && <><option value="all">Match all rules</option><option value="any">Match any rule</option></>}
    </HTMLSelect>
    {group ? <div className="studio-resource-rule-children">
      {(rule.rules || []).map((child, index) => <div key={index}>
        <RuleEditor rule={child} choices={choices} disabled={disabled} depth={depth + 1} path={`${path}.${index + 1}`} onChange={next => onChange({ ...rule, rules: rule.rules.map((r, i) => i === index ? next : r) })}/>
        <Button minimal icon="cross" aria-label={`Remove ${path}.${index + 1}`} disabled={disabled} onClick={() => onChange({ ...rule, rules: rule.rules.filter((_, i) => i !== index) })}>Remove rule</Button>
      </div>)}
      <Button small icon="add" disabled={disabled} onClick={() => onChange({ ...rule, rules: [...(rule.rules || []), blankRule()] })}>Add rule</Button>
    </div> : <div className="studio-resource-rule-selector">
      <InputGroup aria-label={`Search ${path} choices`} leftIcon="search" placeholder="Search provider choices" value={query} onChange={event => setQuery(event.target.value)}/>
      <HTMLSelect aria-label={`${path} value`} value={value} onChange={event => onChange(rule.kind === 'entity' ? { kind: 'entity', entity: event.target.value ? JSON.parse(event.target.value) : { type: '', id: '' } } : { kind: rule.kind, value: event.target.value })}>
        <option value="">Choose from provider</option>
        {optionsWithCurrent.filter(item => optionValue(item) === value || `${item.label} ${item.id || ''} ${item.entity?.type || ''}`.toLowerCase().includes(query.toLowerCase())).map(item => <option key={optionValue(item)} value={optionValue(item)}>{item.label || item.id}{rule.kind === 'entity' ? ` (${item.entity.type}: ${item.entity.id})` : ''}</option>)}
      </HTMLSelect>
      {!options.length && <small>Provider choices are unavailable. Existing rules are preserved.</small>}
    </div>}
  </fieldset>;
}

function completeRule(rule) {
  if (rule.kind === 'all' || rule.kind === 'any') return Array.isArray(rule.rules) && rule.rules.length > 0 && rule.rules.every(completeRule);
  if (rule.kind === 'entity') return Boolean(rule.entity?.type && rule.entity?.id);
  return Boolean(String(rule.value || '').trim());
}
