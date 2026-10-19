import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, ButtonGroup, Callout, Card, InputGroup, Tag } from '@blueprintjs/core';
import { predicateRows } from './predicateCatalog.js';
import { ReaderOutputDialog } from './ReaderOutputDialog.jsx';
import { ReaderParameterDialog } from './ReaderParameterDialog.jsx';
import { ReaderPredicateDialog } from './ReaderPredicateDialog.jsx';

const PAGE_SIZE = 25;
const isServerScope = (item) => item.predicate?.group === 99 || /^param\//i.test(item.source || '');
const inputTabId = (tab) => `reader-input-${tab}-tab`;
const inputPanelId = 'reader-input-details-panel';

function contractInputs(structure) {
  return (structure?.declarations ?? []).map((item) => item?.parameter).filter((item) => item && String(item.source?.kind || '').toLowerCase() !== 'output');
}

function outputColumns(root, path = [], discovered = {}, completeSources = new Set()) {
  if (!root) return [];
  const name = root.namespace || root.name;
  const source = String(root.source?.table || '').toLowerCase();
  const columns = mergeColumns(root.columns, completeSources.has(name.toLowerCase()) ? discovered[source] : []);
  return [...columns.map((column) => ({ column, view: { ...root, columns }, path: [...path, name].join(' / ') })), ...(root.relations ?? []).flatMap((relation) => outputColumns(relation.view, [...path, name], discovered, completeSources))];
}

function mergeColumns(compiled = [], discovered = []) {
  const physical = new Map((discovered ?? []).map((column) => [String(column.source || column.name).toLowerCase(), column]));
  const columns = new Map((discovered ?? []).map((column) => [String(column.name || column.source).toLowerCase(), column]));
  for (const column of compiled ?? []) {
    const source = String(column.source || column.name).toLowerCase();
    const metadata = physical.get(source) || physical.get(source.split('.').at(-1));
    columns.set(String(column.name || column.source).toLowerCase(), { ...(metadata || {}), ...column });
  }
  return [...columns.values()].sort((a, b) => Number(Boolean(b.schema?.primaryKey)) - Number(Boolean(a.schema?.primaryKey)));
}

function tableReference(source) {
  const parts = String(source || '').split('.').map((part) => part.trim()).filter(Boolean);
  return parts.length > 1 ? { schema: parts.slice(0, -1).join('.'), table: parts.at(-1) } : { table: parts[0] || '' };
}

function componentViews(root, path = []) {
  if (!root) return [];
  const name = root.namespace || root.name;
  return [{ view: root, path: [...path, name].join(' / ') }, ...(root.relations ?? []).flatMap((relation) => componentViews(relation.view, [...path, name]))];
}

export function ContractWorkspace({ api, selection, structure, root, connector, onOutputCount, canEdit, activeInputTab, onInputTab, onClear, onApply, onSelectView, onEditOutput, onOutputSQL, onEditingChange, activeOutputTab, onOutputTab, active = true, lookups }) {
  const inputs = useMemo(() => contractInputs(structure), [structure]);
  const outputParameters = useMemo(() => (structure?.declarations ?? []).map(({parameter})=>parameter).filter((item)=>item?.source?.kind==='output'), [structure]);
  const [localOutputTab,setLocalOutputTab] = useState('parameters');
  const outputTab=activeOutputTab??localOutputTab;
  const setOutputTab=onOutputTab??setLocalOutputTab;
  const predicates = useMemo(() => predicateRows(structure), [structure]);
  const completeSources = useMemo(() => {
    const constrained = new Set((structure?.columnContracts ?? []).map((item) => String(item.view || '').toLowerCase()));
    for (const fn of structure?.functions ?? []) {
      if (['output_exclude', 'tag', 'cast'].includes(String(fn.name || '').toLowerCase())) constrained.add(String(fn.args?.[0] || '').split('.')[0].toLowerCase());
    }
    return new Set((structure?.views ?? []).filter((item) => item.sourceProjectionAll && !constrained.has(String(item.name || '').toLowerCase())).map((item) => String(item.name).toLowerCase()));
  }, [structure]);
  const [outputMetadata, setOutputMetadata] = useState({ root: null, columns: {}, loading: false, failures: [] });
  const activeMetadata = outputMetadata.root === root ? outputMetadata : { columns: {}, loading: false, failures: [] };
  const outputs = useMemo(() => outputColumns(root, [], activeMetadata.columns, completeSources), [root, activeMetadata.columns, completeSources]);
  const views = useMemo(() => componentViews(root), [root]);
  const [metadataReload, setMetadataReload] = useState(0);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const [editing, setEditing] = useState(null);
  useEffect(() => { onEditingChange?.(Boolean(editing)); return () => onEditingChange?.(false); }, [Boolean(editing), onEditingChange]);
  const headingRef = useRef(null);
  const isInput = selection?.type === 'input';
  const isViews = selection?.type === 'views';
  const isOutput = selection?.type === 'output';
  const isOutputParameters = isOutput && outputTab === 'parameters';
  const isParameters = isInput || isOutputParameters;
  useEffect(() => {
    if (!active || !isOutput || !root || !api?.getTable || !connector) return;
    const sources = [...new Set(componentViews(root).filter(({ view }) => completeSources.has(String(view.namespace || view.name).toLowerCase())).map(({ view }) => view.source?.table).filter(Boolean))];
    if (!sources.length) { onOutputCount?.(outputColumns(root, [], {}, completeSources).length); return; }
    let cancelled = false;
    setOutputMetadata({ root, columns: {}, loading: true, failures: [] });
    Promise.allSettled(sources.map((source) => api.getTable(connector, tableReference(source)))).then((results) => {
      if (cancelled) return;
      const columns = {};
      const failures = [];
      results.forEach((result, index) => {
        if (result.status !== 'fulfilled') { failures.push(sources[index]); return; }
        columns[sources[index].toLowerCase()] = (result.value?.columns ?? []).map((column) => ({ name: column.name, source: column.name, databaseType: column.type, type: { name: column.type }, groupable: false, schema: column }));
      });
      setOutputMetadata({ root, columns, loading: false, failures });
      onOutputCount?.(failures.length ? null : outputColumns(root, [], columns, completeSources).length);
    });
    return () => { cancelled = true; };
  }, [active, isOutput, root, api, connector, onOutputCount, completeSources, metadataReload]);
  useEffect(() => { setQuery(''); setPage(0); setEditing(null); }, [selection?.type, activeInputTab, outputTab]);
  useEffect(() => { if (!editing && active) headingRef.current?.focus(); }, [selection?.type, editing]);
  const rows = isInput ? activeInputTab === 'predicates' ? predicates : inputs : isOutputParameters ? outputParameters : isViews ? views : outputs;
  const needle = query.trim().toLowerCase();
  const filtered = rows.filter((item) => {
    const searchable = isParameters ? isInput && activeInputTab === 'predicates'
      ? [item.field, item.source, item.predicate?.name, item.predicate?.group, item.view, isServerScope(item) ? 'authorization protected scope' : '', ...(item.predicate?.args ?? [])]
      : [item.name, item.typeExpr, item.source?.kind, item.source?.name, item.description]
      : isViews ? [item.view.name, item.view.namespace, item.view.source?.table, item.path] : [item.column.name, item.column.source, item.column.type?.name, item.column.databaseType, item.path];
    return searchable.join(' ').toLowerCase().includes(needle);
  });
  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const currentPage = Math.min(page, pages - 1);
  const visible = filtered.slice(currentPage * PAGE_SIZE, (currentPage + 1) * PAGE_SIZE);
  const outputVisibility = (item) => {
    const contract = (structure?.columnContracts ?? []).find((entry) => String(entry.view).toLowerCase() === String(item.view.namespace || item.view.name).toLowerCase() && String(entry.column).toLowerCase() === String(item.column.source || item.column.name).toLowerCase());
    return contract?.tags?.json === '-' ? 'Hidden from response' : contract?.tags?.internal === 'true' ? 'Internal' : '';
  };
  const showPredicateScope = predicates.some((item) => item.view || item.predicate?.group);
  const title = isInput ? activeInputTab === 'predicates' ? 'Predicates' : 'Inputs' : isOutputParameters ? 'Output parameters' : isViews ? 'Views' : 'Output columns';
  const onOutputTabKeyDown = (event) => {
    if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return;
    event.preventDefault();
    const next=event.key==='Home'?'parameters':event.key==='End'?'columns':outputTab==='parameters'?'columns':'parameters';
    setOutputTab(next);
    requestAnimationFrame(()=>document.getElementById(`reader-output-${next}-tab`)?.focus());
  };
  const onInputTabKeyDown = (event) => {
    let next;
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') next = activeInputTab === 'parameters' ? 'predicates' : 'parameters';
    else if (event.key === 'Home') next = 'parameters';
    else if (event.key === 'End') next = 'predicates';
    else return;
    event.preventDefault();
    onInputTab(next);
    requestAnimationFrame(() => document.getElementById(inputTabId(next))?.focus());
  };
  return editing ? editing.type === 'output' ? <ReaderOutputDialog key={editing.name || 'new-output'} structure={structure} initialName={editing.name} onClose={()=>setEditing(null)} onApply={onApply}/> : editing.type === 'input' ? <ReaderParameterDialog lookups={lookups} api={api} onApplyPredicate={onApply} embedded formOnly isOpen structure={structure} initialName={editing.name} initialMode={editing.mode} onClose={() => setEditing(null)} onApply={async (operation) => { await onApply(operation); setEditing(null); }}/> : <ReaderPredicateDialog lookups={lookups} embedded formOnly isOpen api={api} structure={structure} initialRowId={editing.id} onClose={() => setEditing(null)} onApply={async (operation) => { await onApply(operation); setEditing(null); }}/> : <Card className="studio-card studio-columns-panel studio-contract-panel" elevation={0}>
      <div className="studio-columns-heading"><div><h2 ref={headingRef} tabIndex={-1}>{title}</h2>{filtered.length > PAGE_SIZE && <span>{filtered.length} total</span>}</div><div className="studio-column-tools"><InputGroup leftIcon="search" aria-label={`Search ${title.toLowerCase()}`} placeholder={isInput ? activeInputTab === 'predicates' ? 'Find parameter, predicate, or view' : 'Find input, source, or type' : isOutputParameters ? 'Find output, kind source, or type' : isViews ? 'Find view, path, or table' : 'Find column, source, or view'} value={query} onChange={(event) => { setQuery(event.target.value); setPage(0); }}/>{canEdit && isInput && (activeInputTab === 'predicates' ? <Button icon="add" onClick={() => setEditing({type:'predicate'})}>Add predicate</Button> : <ButtonGroup><Button icon="add" onClick={() => setEditing({type:'input', mode:'request'})}>Add input</Button><Button onClick={() => setEditing({type:'input', mode:'constant'})}>Add constant</Button></ButtonGroup>)}{canEdit && isOutputParameters && <Button icon="add" onClick={()=>setEditing({type:'output'})}>Add output</Button>}{canEdit && isOutput && !isOutputParameters && root && <Button icon="code" onClick={() => onOutputSQL(root)}>Edit root SQL</Button>}<Button small minimal icon="cross" aria-label="Close contract details" onClick={onClear}/></div></div>
      {isOutput && <div className="studio-contract-tabs" role="tablist" aria-label="Output details"><Button id="reader-output-parameters-tab" role="tab" aria-controls="reader-output-details-panel" tabIndex={isOutputParameters?0:-1} onKeyDown={onOutputTabKeyDown} aria-selected={isOutputParameters} active={isOutputParameters} onClick={()=>setOutputTab('parameters')}>Output parameters</Button><Button id="reader-output-columns-tab" role="tab" aria-controls="reader-output-details-panel" tabIndex={isOutputParameters?-1:0} onKeyDown={onOutputTabKeyDown} aria-selected={!isOutputParameters} active={!isOutputParameters} onClick={()=>setOutputTab('columns')}>Result columns</Button></div>}
      {isOutput && !isOutputParameters && canEdit && <p className="studio-muted">Add fields in the view SQL. Select a field to edit its public name, type, or visibility.</p>}
      {isInput && <div className="studio-contract-tabs" role="tablist" aria-label="Input details"><Button small id={inputTabId('parameters')} role="tab" aria-controls={inputPanelId} tabIndex={activeInputTab === 'parameters' ? 0 : -1} aria-selected={activeInputTab === 'parameters'} active={activeInputTab === 'parameters'} onKeyDown={onInputTabKeyDown} onClick={() => onInputTab('parameters')}>Parameters <Tag minimal>{inputs.length}</Tag></Button><Button small id={inputTabId('predicates')} role="tab" aria-controls={inputPanelId} tabIndex={activeInputTab === 'predicates' ? 0 : -1} aria-selected={activeInputTab === 'predicates'} active={activeInputTab === 'predicates'} onKeyDown={onInputTabKeyDown} onClick={() => onInputTab('predicates')}>Predicates <Tag minimal>{predicates.length}</Tag></Button></div>}
      <div role={isInput || isOutput ? 'tabpanel' : undefined} id={isInput ? inputPanelId : isOutput ? 'reader-output-details-panel' : undefined} aria-labelledby={isInput ? inputTabId(activeInputTab) : isOutput ? `reader-output-${outputTab}-tab` : undefined} tabIndex={isInput || isOutput ? 0 : undefined}>
      {(filtered.length > PAGE_SIZE || needle) && <div className="studio-contract-results" aria-live="polite">{filtered.length ? `${currentPage * PAGE_SIZE + 1}–${Math.min((currentPage + 1) * PAGE_SIZE, filtered.length)} of ${filtered.length}` : '0 results'}</div>}
      {isOutput && activeMetadata.failures.length > 0 && <Callout intent="warning" title="Some connector columns are unavailable" role="alert">Could not inspect {activeMetadata.failures.join(', ')}. Compiled columns remain visible; the output count is incomplete.<Button small minimal intent="warning" icon="refresh" onClick={() => setMetadataReload((value) => value + 1)}>Retry metadata</Button></Callout>}
      {visible.length ? <div className={`studio-view-column-list studio-contract-list ${isInput && activeInputTab === 'predicates' && !showPredicateScope ? 'studio-predicate-list-compact' : ''}`}>{visible.map((item) => isParameters ? isInput && activeInputTab === 'predicates' ? <button type="button" key={item.id} disabled={!canEdit} onClick={() => setEditing({type:'predicate', id:item.id})}><span><span className="studio-predicate-name"><strong>{item.field}</strong>{isServerScope(item) && <Tag minimal intent="warning">Authorization</Tag>}</span><small>{item.source}</small></span><span>{item.kind === 'handler' ? String(item.predicate?.args?.[0] || 'handler').split('.').pop() : item.predicate?.name}</span>{showPredicateScope && <span>{item.view || (item.predicate?.group ? `Group ${item.predicate.group}` : '')}{item.view && item.predicate?.group ? ` · group ${item.predicate.group}` : ''}</span>}<span className="studio-column-edit">{canEdit ? 'Settings' : ''}</span></button> : <button type="button" key={item.name} disabled={!canEdit} onClick={() => setEditing({type:isOutputParameters?'output':'input', name:item.name})}><span><strong>{item.name}</strong><small>{item.source?.kind || 'source'} / {item.source?.name || '—'}</small></span><span>{item.typeExpr || 'inferred'}</span><Tag minimal intent={item.source?.kind === 'const' ? 'primary' : 'none'}>{isOutputParameters ? item.source?.name : item.source?.kind === 'const' ? 'constant' : item.required ? 'required' : 'optional'}</Tag><span className="studio-column-edit">{canEdit ? 'Settings' : ''}</span></button> : isViews ? <button type="button" key={item.path} aria-label={`Open view ${item.path}`} onClick={() => onSelectView(item.view)}><span><strong>{item.view.namespace || item.view.name}</strong><small className="studio-contract-path" title={item.path}>{item.path}</small></span><span>{item.view.source?.table || 'SQL view'}</span><Tag minimal>Level {item.path.split(' / ').length}</Tag><span className="studio-column-edit">Open</span></button> : <button type="button" key={`${item.path}:${item.column.name}`} onClick={() => canEdit ? onEditOutput(item) : onSelectView(item.view)}><span><strong>{item.column.name}</strong>{outputVisibility(item) && <Tag minimal intent="warning">{outputVisibility(item)}</Tag>}<small>{item.column.source || item.column.name}</small></span><span>{item.column.type?.name || item.column.databaseType || 'inferred'}</span><Tag minimal>{item.path}</Tag><span className="studio-column-edit">{canEdit ? 'Settings' : 'View'}</span></button>)}</div> : <div className="studio-empty-compact">{query ? 'No items match this search.' : isOutputParameters ? 'Add an output parameter to define the response.' : isInput ? 'No items in this input catalog.' : isViews ? 'No views are defined.' : activeMetadata.loading ? 'Loading connector columns…' : 'No output columns are exposed by this component.'}</div>}
      {filtered.length > PAGE_SIZE && <div className="studio-column-pages"><ButtonGroup minimal><Button small icon="chevron-left" aria-label="Previous contract page" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}/><span>{currentPage + 1} / {pages}</span><Button small icon="chevron-right" aria-label="Next contract page" disabled={currentPage + 1 >= pages} onClick={() => setPage(currentPage + 1)}/></ButtonGroup></div>}
      </div>
    </Card>;
}
