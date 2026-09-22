import React, { useEffect, useMemo, useState } from 'react';
import { Button, ButtonGroup, Callout, Code, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, HTMLTable, InputGroup, Tag } from '@blueprintjs/core';

// Column authoring uses Datly Reader Builder's atomic setColumnContract
// operation. The browser renders normalized contract metadata and never parses
// or rewrites DQL/tag syntax.
export function ReaderFieldDialog({ isOpen, node, columnContracts = [], onClose, onApply }) {
  const columns = useMemo(() => node?.view?.columns ?? [], [node]);
  const [columnName, setColumnName] = useState('');
  const [role, setRole] = useState('measure');
  const [visibility, setVisibility] = useState('public');
  const [publicName, setPublicName] = useState('');
  const [castType, setCastType] = useState('');
  const [tagName, setTagName] = useState('');
  const [tagValue, setTagValue] = useState('');
  const [advancedTags, setAdvancedTags] = useState({});
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const [showCatalog, setShowCatalog] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const selected = columns.find((item) => columnIdentity(item) === columnName);
  const contract = columnContract(columnContracts, node?.name, columnName);
  const filtered = columns.filter((item) => `${item.name} ${item.source || ''} ${item.type?.name || item.databaseType || ''}`.toLowerCase().includes(query.trim().toLowerCase()));
  const pageSize = 25;
  const pages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pages - 1);
  const visibleColumns = filtered.slice(currentPage * pageSize, currentPage * pageSize + pageSize);

  useEffect(() => {
    if (!isOpen) return;
    const initial = node?.column ? columnIdentity(node.column) : columnIdentity(columns[0]);
    setShowCatalog(!node?.column);
    setColumnName(initial);
    setQuery(''); setPage(0); setSaving(false); setError(''); setTagName(''); setTagValue('');
  }, [isOpen, node, columns]);

  useEffect(() => {
    const next = columnContract(columnContracts, node?.name, columnName);
    setRole(columnRole(selected, next));
    setVisibility(columnVisibility(next));
    setPublicName(publicFieldName(next));
    setCastType(next?.castType || '');
    setAdvancedTags(Object.fromEntries(Object.entries(next?.tags ?? {}).filter(([name]) => !['groupable','internal','json','format'].includes(name))));
  }, [isOpen, columnName, columnContracts, node?.name, selected]);

  const selectColumn = (value) => { setColumnName(value); setError(''); };
  const apply = async (column) => {
    if (!node?.name || !columnName) { setError('Select a compiled view column.'); return false; }
    setSaving(true); setError('');
    try { await onApply({ type: 'setColumnContract', column }); return true; }
    catch (cause) { setError(cause.message); return false; }
    finally { setSaving(false); }
  };
  const saveContract = async () => {
    const tags = { groupable: role === 'dimension' ? 'true' : 'false' };
    if (visibility === 'internal') tags.internal = 'true';
    if (visibility === 'hidden') tags.json = '-';
    if (publicName.trim()) tags.format = `name=${publicName.trim()}`;
    if (await apply({ view: node.name, column: columnName, castType: castType.trim(), tags: { ...advancedTags, ...tags }, removeTags: [...new Set(['groupable', 'internal', 'json', 'format', ...Object.keys(contract?.tags ?? {})])] })) onClose();
  };
  const saveAdvancedTag = () => {
    const name = tagName.trim();
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) { setError('Advanced tag name must be an identifier.'); return; }
    setAdvancedTags((current)=>({...current,[name]:tagValue}));setTagName('');setTagValue('');setError('');
  };
  const removeTag = (name) => setAdvancedTags((current)=>{const next={...current};delete next[name];return next;});

  return <Dialog className="studio-connector-dialog studio-column-dialog" isOpen={isOpen} onClose={onClose} title={`Column contracts · ${node?.label || ''}`} icon="properties" canOutsideClickClose={!saving}>
    <DialogBody>
      <p className="studio-dialog-lead">Edit the selected field’s compiled Datly contract. SQL remains owned by the view; casts and tags remain in the outer projection.</p>
      {error && <Callout intent="danger" role="alert">{error}</Callout>}
      {columns.length === 0 ? <Callout intent="warning" title="No compiled columns">Refresh connector metadata or use an explicit projection so Datly can expose field contracts.</Callout> : <>
        {showCatalog && <><InputGroup leftIcon="search" aria-label="Search columns" placeholder="Find 100+ columns by name, source, or type" value={query} onChange={(event) => { setQuery(event.target.value); setPage(0); }}/>
        <div className="studio-column-table-wrap"><HTMLTable striped className="studio-data-table studio-field-role-table"><thead><tr><th>Field</th><th>Type</th><th>Contract</th></tr></thead><tbody>{visibleColumns.map((column) => {
          const itemContract = columnContract(columnContracts, node?.name, columnIdentity(column));
          return <tr key={columnIdentity(column)} className={columnName === columnIdentity(column) ? 'studio-selected-row' : ''}><td><button type="button" className="studio-cell-button" onClick={() => selectColumn(columnIdentity(column))}><strong>{column.name}</strong><small className="studio-tree-meta">{column.source || column.name}</small></button></td><td>{column.type?.name || column.databaseType || 'inferred'}</td><td><Tag minimal intent={columnRole(column, itemContract) === 'dimension' ? 'primary' : 'none'}>{columnRole(column, itemContract)}</Tag> {columnVisibility(itemContract) !== 'public' && <Tag minimal>{columnVisibility(itemContract)}</Tag>}</td></tr>;
        })}</tbody></HTMLTable></div>
        {filtered.length > pageSize && <div className="studio-column-pages"><span>{currentPage * pageSize + 1}–{Math.min((currentPage + 1) * pageSize, filtered.length)} of {filtered.length}</span><ButtonGroup minimal><Button small icon="chevron-left" aria-label="Previous column page" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}/><span>{currentPage + 1} / {pages}</span><Button small icon="chevron-right" aria-label="Next column page" disabled={currentPage + 1 >= pages} onClick={() => setPage(currentPage + 1)}/></ButtonGroup></div>}
        </>}
        <section className="studio-column-contract" aria-label="Selected column contract">
          <div className="studio-column-contract-heading"><div><h3>{selected?.name || columnName}</h3><p><Code>{node?.name}.{columnName}</Code> · {selected?.databaseType || 'database type inferred'}</p></div><div><Tag minimal>{contract?.castType || selected?.type?.name || 'inferred Go type'}</Tag>{node?.column&&<Button small minimal icon={showCatalog?'chevron-up':'list'} onClick={()=>setShowCatalog((value)=>!value)}>{showCatalog?'Hide columns':'Choose another column'}</Button>}</div></div>
          <div className="studio-form-grid"><FormGroup label="Output visibility" labelFor="column-visibility" helperText="Internal remains SQL-mapped; hidden is omitted from JSON."><HTMLSelect id="column-visibility" value={visibility} onChange={(event) => setVisibility(event.target.value)} fill disabled={saving}><option value="public">Public output</option><option value="internal">Internal backing field</option><option value="hidden">Hidden from JSON</option></HTMLSelect></FormGroup><FormGroup label="Cube role" labelFor="column-role"><HTMLSelect id="column-role" value={role} onChange={(event) => setRole(event.target.value)} fill disabled={saving}><option value="dimension">Dimension</option><option value="measure">Measure / field</option></HTMLSelect></FormGroup></div>
          <div className="studio-form-grid"><FormGroup label="Go cast" labelFor="column-cast" helperText="Clear to remove the explicit CAST."><InputGroup id="column-cast" value={castType} onChange={(event) => setCastType(event.target.value)} placeholder="model.Money or string" disabled={saving}/></FormGroup><FormGroup label="Public field name" labelFor="column-public-name" helperText="Uses Datly format metadata and component casing."><InputGroup id="column-public-name" value={publicName} onChange={(event) => setPublicName(event.target.value)} placeholder="CustomerName" disabled={saving}/></FormGroup></div>
        </section>
        <section className="studio-column-authoring-section studio-column-authoring-advanced"><h3>Advanced metadata tags</h3><p>Use registered Datly/SQLX tag keys. Changes are staged with the contract and applied only when you choose Save column contract.</p><div className="studio-current-tags">{Object.entries(advancedTags).map(([name, value]) => <Tag key={name} minimal onRemove={saving ? undefined : () => removeTag(name)}>{name}: {value || 'empty'}</Tag>)}</div><div className="studio-function-form"><InputGroup aria-label="Advanced tag name" value={tagName} onChange={(event) => setTagName(event.target.value)} placeholder="codec" disabled={saving}/><InputGroup aria-label="Advanced tag value" value={tagValue} onChange={(event) => setTagValue(event.target.value)} placeholder="registered codec options" disabled={saving}/><Button icon="tag" disabled={saving} onClick={saveAdvancedTag}>Stage tag</Button></div></section>
      </>}
    </DialogBody>
    <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>Cancel</Button><Button intent="primary" icon="floppy-disk" loading={saving} disabled={!columnName} onClick={saveContract}>Save column contract</Button></>}/>
  </Dialog>;
}

function columnIdentity(column) { return String(column?.source || column?.name || ''); }
function columnContract(contracts, view, column) { return (contracts ?? []).find((item) => String(item.view).toLowerCase() === String(view).toLowerCase() && String(item.column).toLowerCase() === String(column).toLowerCase()); }
function columnRole(column, contract) { if (contract?.tags?.groupable === 'true' || column?.groupable === true) return 'dimension'; return 'measure'; }
function columnVisibility(contract) { if (contract?.tags?.internal === 'true') return 'internal'; if (contract?.tags?.json === '-') return 'hidden'; return 'public'; }
function publicFieldName(contract) { const value = String(contract?.tags?.format || ''); return value.startsWith('name=') ? value.slice(5) : ''; }
