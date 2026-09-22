import React, { useMemo, useState } from 'react';
import { Button, ButtonGroup, Code, HTMLTable, Tag } from '@blueprintjs/core';
import { formatCell, isStructured, nestedFields, previewCollections, rowIdentity, scalarColumns } from './nestedDataGrid.js';

export function NestedDataGrid({ data }) {
  const collections = useMemo(() => previewCollections(data), [data]);
  const residual = useMemo(() => {
    if (!isStructured(data) || Array.isArray(data)) return [];
    return Object.entries(data).filter(([, value]) => !Array.isArray(value));
  }, [data]);
  if (collections.length === 0) return <NestedValue name="Result" value={data} depth={0}/>;
  return <div className="studio-result-grid">
    {collections.map((collection) => <CollectionTable key={collection.name} name={collection.name} rows={collection.rows} depth={0}/>)}
    {residual.length > 0 && <div className="studio-result-derived"><h3>Derived outputs</h3>{residual.map(([name, value]) => <NestedValue key={name} name={name} value={value} depth={0}/>)}</div>}
  </div>;
}

export function ResultPreview({ data }) {
  const [mode, setMode] = useState('table');
  return <div className="studio-result-preview">
    <div className="studio-result-mode" role="group" aria-label="Preview format">
      <ButtonGroup minimal>
        <Button small icon="th" active={mode === 'table'} aria-pressed={mode === 'table'} onClick={() => setMode('table')}>Table</Button>
        <Button small icon="code" active={mode === 'json'} aria-pressed={mode === 'json'} onClick={() => setMode('json')}>JSON</Button>
      </ButtonGroup>
    </div>
    {mode === 'table' ? <NestedDataGrid data={data}/> : <pre className="studio-preview-source studio-preview-json">{JSON.stringify(data, null, 2)}</pre>}
  </div>;
}

function CollectionTable({ name, rows, depth }) {
  const [expanded, setExpanded] = useState(() => new Set());
  const columns = useMemo(() => scalarColumns(rows), [rows]);
  const toggle = (identity) => setExpanded((current) => {
    const next = new Set(current);
    if (next.has(identity)) next.delete(identity); else next.add(identity);
    return next;
  });
  return <section className={`studio-nested-collection studio-nested-depth-${Math.min(depth, 3)}`}>
    <div className="studio-result-heading"><div><h3>{name}</h3><span>{rows.length} {rows.length === 1 ? 'row' : 'rows'}</span></div>{depth === 0 && <Tag minimal intent="primary">Datly result</Tag>}</div>
    {rows.length === 0 ? <div className="studio-result-empty">No rows returned.</div> : <div className="studio-table-wrap"><HTMLTable className="studio-data-table studio-result-table" striped interactive>
      <thead><tr><th aria-label="Expand embedded subtables"></th>{columns.map((column) => <th key={column}>{column}</th>)}</tr></thead>
      <tbody>{rows.map((row, index) => {
        const identity = `${rowIdentity(row, index)}:${index}`;
        const nested = nestedFields(row);
        const open = expanded.has(identity);
        return <React.Fragment key={identity}>
          <tr>
            <td><Button minimal small icon={open ? 'chevron-down' : 'chevron-right'} disabled={nested.length === 0} aria-label={`${open ? 'Collapse' : 'Expand'} row ${index + 1}`} aria-expanded={open} onClick={() => toggle(identity)}/></td>
            {columns.map((column) => <td key={column}>{formatCell(row?.[column])}</td>)}
          </tr>
          {open && <tr className="studio-expanded-row"><td colSpan={columns.length + 1}><div className="studio-expanded-content">{nested.map((field) => <NestedValue key={field.name} name={field.name} value={field.value} depth={depth + 1}/>)}</div></td></tr>}
        </React.Fragment>;
      })}</tbody>
    </HTMLTable></div>}
  </section>;
}

function NestedValue({ name, value, depth }) {
  if (depth > 4) return <div className="studio-nested-value"><strong>{name}</strong><Code>Nested depth limit reached</Code></div>;
  if (Array.isArray(value)) {
    if (value.every((item) => !isStructured(item))) return <div className="studio-nested-value"><strong>{name}</strong><span>{value.map(formatCell).join(', ') || '—'}</span></div>;
    return <CollectionTable name={name} rows={value} depth={depth}/>;
  }
  if (isStructured(value)) {
    const entries = Object.entries(value);
    return <section className="studio-nested-object"><div className="studio-result-heading"><div><h3>{name}</h3><span>object</span></div></div><dl>{entries.map(([field, item]) => <React.Fragment key={field}><dt>{field}</dt><dd>{isStructured(item) ? <NestedValue name={field} value={item} depth={depth + 1}/> : formatCell(item)}</dd></React.Fragment>)}</dl></section>;
  }
  return <div className="studio-nested-value"><strong>{name}</strong><span>{formatCell(value)}</span></div>;
}
