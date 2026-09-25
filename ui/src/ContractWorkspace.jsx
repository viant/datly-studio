import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, ButtonGroup, Card, InputGroup, Tag } from '@blueprintjs/core';
import { predicateRows } from './predicateCatalog.js';
import { ReaderParameterDialog } from './ReaderParameterDialog.jsx';
import { ReaderPredicateDialog } from './ReaderPredicateDialog.jsx';

const PAGE_SIZE = 25;

function contractInputs(structure) {
  return (structure?.declarations ?? []).map((item) => item?.parameter).filter((item) => item && !['output', 'component', 'view'].includes(String(item.source?.kind || '').toLowerCase()));
}

function outputColumns(root, path = []) {
  if (!root) return [];
  const name = root.namespace || root.name;
  return [...(root.columns ?? []).map((column) => ({ column, view: root, path: [...path, name].join(' / ') })), ...(root.relations ?? []).flatMap((relation) => outputColumns(relation.view, [...path, name]))];
}

function componentViews(root, path = []) {
  if (!root) return [];
  const name = root.namespace || root.name;
  return [{ view: root, path: [...path, name].join(' / ') }, ...(root.relations ?? []).flatMap((relation) => componentViews(relation.view, [...path, name]))];
}

export function ContractWorkspace({ api, selection, structure, root, canEdit, activeInputTab, onInputTab, onClear, onApply, onSelectView }) {
  const inputs = useMemo(() => contractInputs(structure), [structure]);
  const predicates = useMemo(() => predicateRows(structure), [structure]);
  const outputs = useMemo(() => outputColumns(root), [root]);
  const views = useMemo(() => componentViews(root), [root]);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const [editing, setEditing] = useState(null);
  const headingRef = useRef(null);
  const isInput = selection?.type === 'input';
  const isViews = selection?.type === 'views';
  useEffect(() => { setQuery(''); setPage(0); setEditing(null); }, [selection?.type, activeInputTab]);
  useEffect(() => { if (!editing) headingRef.current?.focus(); }, [selection?.type, editing]);
  const rows = isInput ? activeInputTab === 'predicates' ? predicates : inputs : isViews ? views : outputs;
  const needle = query.trim().toLowerCase();
  const filtered = rows.filter((item) => {
    const searchable = isInput ? activeInputTab === 'predicates'
      ? [item.field, item.source, item.predicate?.name, item.view, ...(item.predicate?.args ?? [])]
      : [item.name, item.typeExpr, item.source?.kind, item.source?.name, item.description]
      : isViews ? [item.view.name, item.view.namespace, item.view.source?.table, item.path] : [item.column.name, item.column.source, item.column.type?.name, item.column.databaseType, item.path];
    return searchable.join(' ').toLowerCase().includes(needle);
  });
  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const currentPage = Math.min(page, pages - 1);
  const visible = filtered.slice(currentPage * PAGE_SIZE, (currentPage + 1) * PAGE_SIZE);
  const title = isInput ? activeInputTab === 'predicates' ? 'Predicates' : 'Inputs' : isViews ? 'Views' : 'Output columns';
  return editing ? editing.type === 'input' ? <ReaderParameterDialog embedded formOnly isOpen structure={structure} initialName={editing.name} initialMode={editing.mode} onClose={() => setEditing(null)} onApply={async (operation) => { await onApply(operation); setEditing(null); }}/> : <ReaderPredicateDialog embedded formOnly isOpen api={api} structure={structure} initialRowId={editing.id} onClose={() => setEditing(null)} onApply={async (operation) => { await onApply(operation); setEditing(null); }}/> : <Card className="studio-card studio-columns-panel studio-contract-panel" elevation={0}>
      <div className="studio-columns-heading"><div><h2 ref={headingRef} tabIndex={-1}>{title}</h2>{filtered.length > PAGE_SIZE && <span>{filtered.length} total</span>}</div><div className="studio-column-tools"><InputGroup leftIcon="search" aria-label={`Search ${title.toLowerCase()}`} placeholder={isInput ? activeInputTab === 'predicates' ? 'Find parameter, predicate, or view' : 'Find input, source, or type' : isViews ? 'Find view, path, or table' : 'Find column, source, or view'} value={query} onChange={(event) => { setQuery(event.target.value); setPage(0); }}/>{canEdit && isInput && (activeInputTab === 'predicates' ? <Button icon="add" onClick={() => setEditing({type:'predicate'})}>Add predicate</Button> : <ButtonGroup><Button icon="add" onClick={() => setEditing({type:'input', mode:'request'})}>Add input</Button><Button onClick={() => setEditing({type:'input', mode:'constant'})}>Add constant</Button></ButtonGroup>)}<Button small minimal icon="cross" aria-label="Close contract details" onClick={onClear}/></div></div>
      {isInput && <div className="studio-contract-tabs" role="tablist" aria-label="Input details"><Button small role="tab" aria-selected={activeInputTab === 'parameters'} active={activeInputTab === 'parameters'} onClick={() => onInputTab('parameters')}>Parameters <Tag minimal>{inputs.length}</Tag></Button><Button small role="tab" aria-selected={activeInputTab === 'predicates'} active={activeInputTab === 'predicates'} onClick={() => onInputTab('predicates')}>Predicates <Tag minimal>{predicates.length}</Tag></Button></div>}
      {(filtered.length > PAGE_SIZE || needle) && <div className="studio-contract-results" aria-live="polite">{filtered.length ? `${currentPage * PAGE_SIZE + 1}–${Math.min((currentPage + 1) * PAGE_SIZE, filtered.length)} of ${filtered.length}` : '0 results'}</div>}
      {visible.length ? <div className="studio-view-column-list studio-contract-list">{visible.map((item) => isInput ? activeInputTab === 'predicates' ? <button type="button" key={item.id} disabled={!canEdit} onClick={() => setEditing({type:'predicate', id:item.id})}><span><strong>{item.field}</strong><small>{item.source}</small></span><span>{item.predicate?.name}</span><span>{item.view || 'Unassigned'} · group {item.predicate?.group}</span><span className="studio-column-edit">{canEdit ? 'Settings' : ''}</span></button> : <button type="button" key={item.name} disabled={!canEdit} onClick={() => setEditing({type:'input', name:item.name})}><span><strong>{item.name}</strong><small>{item.source?.kind || 'source'} / {item.source?.name || '—'}</small></span><span>{item.typeExpr || 'inferred'}</span><Tag minimal intent={item.source?.kind === 'const' ? 'primary' : 'none'}>{item.source?.kind === 'const' ? 'constant' : item.required ? 'required' : 'optional'}</Tag><span className="studio-column-edit">{canEdit ? 'Settings' : ''}</span></button> : isViews ? <button type="button" key={item.path} aria-label={`Open view ${item.path}`} onClick={() => onSelectView(item.view)}><span><strong>{item.view.namespace || item.view.name}</strong><small className="studio-contract-path" title={item.path}>{item.path}</small></span><span>{item.view.source?.table || 'SQL view'}</span><Tag minimal>Level {item.path.split(' / ').length}</Tag><span className="studio-column-edit">Open</span></button> : <button type="button" key={`${item.path}:${item.column.name}`} onClick={() => onSelectView(item.view)}><span><strong>{item.column.name}</strong><small>{item.column.source || item.column.name}</small></span><span>{item.column.type?.name || item.column.databaseType || 'inferred'}</span><Tag minimal>{item.path}</Tag><span className="studio-column-edit">View</span></button>)}</div> : <div className="studio-empty-compact">{query ? 'No items match this search.' : isInput ? 'No items in this input catalog.' : isViews ? 'No views are defined.' : 'No output columns are exposed by this component.'}</div>}
      {filtered.length > PAGE_SIZE && <div className="studio-column-pages"><ButtonGroup minimal><Button small icon="chevron-left" aria-label="Previous contract page" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}/><span>{currentPage + 1} / {pages}</span><Button small icon="chevron-right" aria-label="Next contract page" disabled={currentPage + 1 >= pages} onClick={() => setPage(currentPage + 1)}/></ButtonGroup></div>}
    </Card>;
}
