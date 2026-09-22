import React, { useEffect, useMemo, useState } from 'react';
import { Button, Callout, Card, Code, HTMLSelect, HTMLTable, InputGroup, Spinner, Tag } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';
import { ResultPreview } from './NestedDataGrid.jsx';
import { SchemaRootDialog } from './SchemaRootDialog.jsx';
import { SchemaSubviewDialog } from './SchemaSubviewDialog.jsx';

export function SchemaBrowser({ api, onOpenBuilder }) {
  const [connectors, setConnectors] = useState([]);
  const [connector, setConnector] = useState('');
  const [schemas, setSchemas] = useState([]);
  const [schema, setSchema] = useState('');
  const [tables, setTables] = useState([]);
  const [query, setQuery] = useState('');
  const [selected, setSelected] = useState(null);
  const [sqlSource, setSQLSource] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [sqlResult, setSQLResult] = useState(null);
  const [testingSQL, setTestingSQL] = useState(false);
  const [rootDialogOpen, setRootDialogOpen] = useState(false);
  const [subviewDialogOpen, setSubviewDialogOpen] = useState(false);

  useEffect(() => { api.listConnectors({ status: 'active' }).then((page) => { const items = page?.items ?? []; setConnectors(items); setConnector(items[0]?.name ?? ''); }).catch((cause) => setError(cause.message)).finally(() => setLoading(false)); }, [api]);
  useEffect(() => { if (!connector) return; setLoading(true); setError(''); setSelected(null); setSQLResult(null); api.listSchemas(connector).then((result) => { const items=result?.items??[]; setSchemas(items); setSchema(''); }).catch((cause)=>setError(cause.message)).finally(()=>setLoading(false)); }, [api,connector]);
  useEffect(() => { if (!connector) return; const timer=setTimeout(()=>{ setLoading(true); api.listTables(connector,{schema,query,limit:200}).then((result)=>setTables(result?.items??[])).catch((cause)=>setError(cause.message)).finally(()=>setLoading(false)); },180); return()=>clearTimeout(timer); },[api,connector,schema,query]);
  const openTable = async (table) => { setError(''); setSQLResult(null); try { const detail=await api.getTable(connector,{schema:table.schema||schema,table:table.name}); setSelected(detail); setSQLSource(starterSQL(detail.table)); } catch(cause){setError(cause.message);} };
  const testSQL = async () => { if(!sqlSource.trim())return;setTestingSQL(true);setError('');setSQLResult(null);try{setSQLResult(await api.testSQL(connector,{sql:sqlSource.trim(),limit:50}));}catch(cause){setError(cause.message);}finally{setTestingSQL(false);} };
  const closeSelected = () => { setSelected(null); setSQLSource(''); setSQLResult(null); setRootDialogOpen(false); setSubviewDialogOpen(false); };
  const title = selected?.table?.name ?? 'Select a table';
  const selectedSource = selected ? {...selected.table,columns:selected.columns} : null;
  return <><main className="studio-workspace studio-schema-workspace">
    <div className="studio-schema-header"><div><h1 className="studio-page-heading">Schema Browser</h1><p className="studio-page-description">Explore authorized connector metadata and use tables as Datly view sources.</p></div><Tag minimal intent="primary">SQLX metadata</Tag></div>
    {error && <Callout intent="danger" title="Schema Browser request failed" role="alert">{error}</Callout>}
    <div className="studio-schema-layout">
      <aside className="studio-schema-explorer">
        <div className="studio-schema-controls"><HTMLSelect fill value={connector} onChange={(event)=>setConnector(event.target.value)} aria-label="Connector">{connectors.map((item)=><option key={item.name}>{item.name}</option>)}</HTMLSelect><HTMLSelect fill value={schema} onChange={(event)=>setSchema(event.target.value)} aria-label="Schema"><option value="">Default schema</option>{schemas.map((item)=><option key={`${item.catalog}:${item.name}`} value={item.name}>{item.name||'Default schema'}</option>)}</HTMLSelect><InputGroup leftIcon="search" placeholder="Find tables or views" value={query} onChange={(event)=>setQuery(event.target.value)} /></div>
        <div className="studio-schema-list" role="tree" aria-label="Database objects">{loading && <Spinner size={20}/>} {!loading&&tables.length===0&&<div className="studio-empty-compact">No matching tables.</div>}{tables.map((table)=><button type="button" role="treeitem" className={`studio-schema-item ${selected?.table?.name===table.name?'selected':''}`} key={`${table.schema}:${table.name}`} onClick={()=>openTable(table)}><span className="studio-schema-icon">▦</span><span><strong>{table.name}</strong><small>{table.schema||schema||'default'} · {table.type||'TABLE'}</small></span></button>)}</div>
      </aside>
      <section className="studio-schema-main">
        <div className="studio-schema-tabs">{selected && <div className="studio-schema-tab selected"><button type="button" title={title}>{title}</button><button type="button" className="studio-schema-tab-close" aria-label={`Close ${title} table`} title={`Close ${title}`} onClick={closeSelected}>×</button></div>}<button type="button" className="studio-schema-tab-new" title="Open another table" aria-label="Open another table" onClick={closeSelected}>＋</button></div>
        {!selected ? <Card className="studio-card studio-schema-empty" elevation={0}><h2>Choose a database object</h2><p>Inspect columns and relationships, then add it to a Datly component as a root view or subview.</p></Card> : <>
          <Card className="studio-card studio-schema-editor" elevation={0}><div className="studio-panel-heading"><div><h2>{selected.table.name}</h2><p className="studio-muted">Starter SQL for the selected Datly view.</p></div><Code>{connector}</Code></div><Editor ariaLabel={`${selected.table.name} SQL source`} value={sqlSource} onChange={(value)=>{setSQLSource(value);setSQLResult(null);}} language="sql" height="240px"/><div className="studio-schema-actions"><Button icon="play" loading={testingSQL} onClick={testSQL}>Test SQL</Button><Button intent="primary" icon="new-object" onClick={()=>setRootDialogOpen(true)}>Add as root view</Button><Button icon="git-branch" onClick={()=>setSubviewDialogOpen(true)}>Add as subview</Button></div></Card>
          {sqlResult&&<Card className="studio-card studio-schema-columns" elevation={0}><div className="studio-panel-heading"><div><h2>SQL test result</h2><p className="studio-muted">Bounded transient Datly reader execution.</p></div><Code>{formatDuration(sqlResult.duration)}</Code></div><ResultPreview data={sqlResult.data}/></Card>}
          <Card className="studio-card studio-schema-columns" elevation={0}><h2>Columns</h2><HTMLTable striped className="studio-data-table"><thead><tr><th>Name</th><th>Type</th><th>Constraints</th><th>Reference</th></tr></thead><tbody>{selected.columns.map((column)=><tr key={column.name}><td><strong>{column.name}</strong></td><td>{column.type}</td><td>{column.primaryKey&&<Tag minimal intent="primary">PK</Tag>} {column.unique&&<Tag minimal>unique</Tag>} {column.nullable?<span className="studio-muted">nullable</span>:<span>required</span>}</td><td>{column.referenceTable?<code>{column.referenceTable}.{column.referenceColumn}</code>:''}</td></tr>)}</tbody></HTMLTable></Card>
        </>}
      </section>
    </div>
  </main><SchemaRootDialog api={api} isOpen={rootDialogOpen} connector={connector} table={selected?.table} sql={sqlSource} onClose={()=>setRootDialogOpen(false)} onCreated={onOpenBuilder}/><SchemaSubviewDialog api={api} isOpen={subviewDialogOpen} connector={connector} table={selectedSource} sql={sqlSource} onClose={()=>setSubviewDialogOpen(false)} onApplied={onOpenBuilder}/></>;
}

function starterSQL(table){const qualified=[table.schema,table.name].filter(Boolean).join('.');return `SELECT *\nFROM ${qualified}`;}
function formatDuration(value){const nanoseconds=Number(value);if(!Number.isFinite(nanoseconds))return 'completed';if(nanoseconds<1e6)return `${Math.round(nanoseconds/1e3)}µs`;return `${(nanoseconds/1e6).toFixed(1)}ms`;}
