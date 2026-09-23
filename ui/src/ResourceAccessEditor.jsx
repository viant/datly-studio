import React, { useEffect, useRef, useState } from 'react';
import { Button, Callout, FormGroup, HTMLSelect, InputGroup, Spinner, Tag } from '@blueprintjs/core';
import './resourceAccess.css';

const publicActions = new Set(['discover', 'describe', 'execute', 'retrieve']);
const blankRule = () => ({ kind: 'role', value: '' });
const clone = value => JSON.parse(JSON.stringify(value));

// Resource types and selector catalogs belong to the embedding application.
// This editor never interprets JWTs or computes an authoritative access decision.
export function ResourceAccessEditor({ api, resource, actions, choices = {}, readOnly = false }) {
  const [document, setDocument] = useState(null);
  const [draft, setDraft] = useState(null);
  const [action, setAction] = useState(actions[0]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState(null);
  const [saved, setSaved] = useState(false);
  const [editorContext, setEditorContext] = useState(null);
  const sequence = useRef(0);
  const resourceKey = JSON.stringify(resource);
  const load = async () => {
    const ticket = ++sequence.current;
    setLoading(true); setError(null); setSaved(false);
    try {
      const [value, context] = await Promise.all([
        api.getResourceAccess(resource),
        api.getResourceAccessContext ? api.getResourceAccessContext(resource) : Promise.resolve(null),
      ]);
      if (ticket !== sequence.current) return;
      setDocument(value); setDraft(clone(value)); setEditorContext(context);
    } catch (cause) { if (ticket === sequence.current) setError(cause); }
    finally { if (ticket === sequence.current) setLoading(false); }
  };
  useEffect(() => {
    setDocument(null); setDraft(null); setAction(actions[0]); setSaving(false); load();
    return () => { sequence.current++; };
  }, [api, resourceKey]);
  const update = policy => {
    setDraft(current => ({ ...current, policies: { ...current.policies, [action]: policy } }));
    setSaved(false);
  };
  const save = async () => {
    const ticket = sequence.current;
    setSaving(true); setError(null);
    try {
      const result = await api.replaceResourceAccess(draft);
      if (ticket !== sequence.current) return;
      setDocument(result); setDraft(clone(result)); setSaved(true);
    } catch (cause) { if (ticket === sequence.current) setError(cause); }
    finally { if (ticket === sequence.current) setSaving(false); }
  };
  const dirty = document && JSON.stringify(document) !== JSON.stringify(draft);
  const policy = draft?.policies?.[action];
  const cannotManage = readOnly || editorContext?.canManage === false;
  const disabled = cannotManage || saving || loading;
  const providerChoices = editorContext?.choices || choices;
  const conflict = error?.code === 'conflict';
  return <section className="studio-resource-access" aria-label="Resource access">
    <header className="studio-resource-access-heading">
      <div><h2>Access permissions</h2><p>{resource.kind} · <strong>{resource.id}</strong> · {resource.tenant} · version {resource.version}</p></div>
      {document && <Tag minimal>Policy revision {document.revision}</Tag>}
    </header>
    {error && <Callout intent={conflict ? 'warning' : 'danger'} role="alert" title={conflict ? 'Permissions changed elsewhere' : 'Access request failed'}>
      {conflict ? 'Your edits are preserved. Reload the saved policy to review the latest permissions before making changes again.' : error.message}
      <Button minimal onClick={load} disabled={saving}>{dirty ? 'Discard edits and reload' : 'Retry'}</Button>
    </Callout>}
    {loading && <div role="status" className="studio-resource-access-loading"><Spinner size={22}/> Loading permissions…</div>}
    {!loading && draft && <div className="studio-resource-access-layout">
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
            if (!mode) { const policies = { ...draft.policies }; delete policies[action]; setDraft({ ...draft, policies }); }
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
      <Button intent="primary" icon="floppy-disk" disabled={disabled || !dirty || conflict} loading={saving} onClick={save}>Save permissions</Button>
    </footer>}
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
