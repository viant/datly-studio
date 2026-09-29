import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Callout, Checkbox, FormGroup, InputGroup } from '@blueprintjs/core';

// Wire names retain the canonical column spelling; Datly lowercases only the
// first letter when compiling the derived cube's public boolean fields.
const wireName = (value) => String(value || '').replace(/^./, (letter) => letter.toLowerCase());
export function ReaderCubePreview({ structure, pending, onRun, onClose }) {
  const heading = useRef(null);
  useEffect(()=>{heading.current?.focus();},[]);
  const columns = structure?.component?.rootView?.columns ?? [];
  const [selected, setSelected] = useState({});
  const [query, setQuery] = useState('');
  const [values, setValues] = useState({});
  const [filterQuery, setFilterQuery] = useState('');
  const [error, setError] = useState('');
  const filters = useMemo(() => (structure?.declarations ?? []).map((item) => item.parameter).filter((item) => item?.source?.kind === 'query'), [structure]);
  const choices = columns.filter((column)=>typeof column.groupable==='boolean').filter((column) => `${column.name} ${column.source || ''}`.toLowerCase().includes(query.toLowerCase()));
  const count = Object.values(selected).filter(Boolean).length;
  const unresolved = columns.filter((column)=>typeof column.groupable!=='boolean').length;
  const filterMatches = filters.filter((item)=>item.name.toLowerCase().includes(filterQuery.toLowerCase()));
  const missingMeasures = !columns.some((column)=>column.groupable===false);

  const run = async () => {
    const dimensions = {}, measures = {}, inputFilters = {};
    for (const column of columns.filter((column)=>typeof column.groupable==='boolean')) (column.groupable === true ? dimensions : measures)[wireName(column.name)] = Boolean(selected[column.name]);
    try {
      for (const parameter of filters) {
        const value = values[parameter.name];
        if (value == null || value === '') continue;
        const converted = convertFilter(parameter, value);
        inputFilters[wireName(parameter.name)] = converted;
      }
      setError(''); await onRun({dimensions,measures,filters:inputFilters});
    } catch (cause) {setError(cause.message);}
  };
  return <section className="studio-cube-preview" aria-label="Cube preview selection">
    <div className="studio-panel-heading"><h2 ref={heading} tabIndex={-1}>Cube preview</h2><Button minimal icon="cross" aria-label="Close cube preview selection" disabled={pending} onClick={onClose}/></div>
    {missingMeasures && <Callout intent="warning">No measures are available in the resolved cube contract. Configure measure roles before running the preview.</Callout>}
    {error && <Callout intent="danger" role="alert">{error}</Callout>}
    {unresolved>0&&<p className="studio-muted">{unresolved} fields need a dimension or measure role.</p>}
    <InputGroup leftIcon="search" aria-label="Find cube fields" placeholder="Find a dimension or measure" value={query} onChange={(event)=>setQuery(event.target.value)}/>
    <div className="studio-cube-field-groups">{['Dimensions','Measures'].map((role)=><fieldset key={role}><legend>{role}</legend><div className="studio-cube-field-list">{choices.filter((column)=>(column.groupable===true)===(role==='Dimensions')).map((column)=><Checkbox key={column.name} label={column.name} checked={Boolean(selected[column.name])} disabled={pending} onChange={(event)=>setSelected({...selected,[column.name]:event.target.checked})}/>)}{!choices.some((column)=>(column.groupable===true)===(role==='Dimensions'))&&<p className="studio-muted">No matching {role.toLowerCase()}.</p>}</div></fieldset>)}</div>
    {filters.length>0&&<details><summary>Filters</summary><InputGroup aria-label="Find cube filters" placeholder="Find a filter" value={filterQuery} onChange={(event)=>setFilterQuery(event.target.value)}/><p className="studio-muted">Showing {Math.min(filterMatches.length,10)} of {filterMatches.length} filters{filterMatches.length>10?' · Refine search to find another filter':''}</p>{filterMatches.slice(0,10).map((parameter)=><FormGroup key={parameter.name} label={parameter.name} labelFor={`cube-filter-${parameter.name}`} helperText={!supportsFilter(parameter)?`Filter type ${parameter.typeExpr||'unknown'} needs a structured editor`:String(parameter.typeExpr||'').startsWith('[]')?'Comma-separated values':undefined}><InputGroup id={`cube-filter-${parameter.name}`} value={values[parameter.name]||''} disabled={pending||!supportsFilter(parameter)} onChange={(event)=>setValues({...values,[parameter.name]:event.target.value})}/></FormGroup>)}</details>}
    <div className="studio-cube-preview-actions"><span aria-live="polite">{count} selected</span><Button intent="primary" icon="play" disabled={count===0||missingMeasures||pending} loading={pending} onClick={run}>Run cube preview</Button></div>
  </section>;
}

function convertFilter(parameter, value) {
  const type = String(parameter.typeExpr || '').replace(/^\*/, '');
  const array = type.startsWith('[]');
  const scalar = array ? type.slice(2) : type;
  const items = array ? value.split(',').map((item)=>item.trim()) : [value];
  if (items.some((item)=>item==='')) throw new Error(`${parameter.name} contains an empty value.`);
  const converted = items.map((item)=>{
    if (scalar==='string') return item;
    if (scalar==='bool') {if(!['true','false'].includes(item))throw new Error(`${parameter.name} requires true or false.`);return item==='true';}
    if (/^(u?int)(8|16|32|64)?$/.test(scalar)) {
      if (!/^-?\d+$/.test(item)) throw new Error(`${parameter.name} requires whole numbers.`);
      const number=Number(item),bits=Number(scalar.match(/\d+/)?.[0]||64),unsigned=scalar.startsWith('uint');
      const min=unsigned?0:-(2**(bits-1)),max=unsigned?2**bits-1:2**(bits-1)-1;
      if(!Number.isSafeInteger(number)||number<min||number>max)throw new Error(`${parameter.name} is outside the supported ${scalar} range.`);
      return number;
    }
    if (/^float(32|64)?$/.test(scalar)) {const number=Number(item);if(!Number.isFinite(number))throw new Error(`${parameter.name} requires finite numbers.`);return number;}
    throw new Error(`${parameter.name} uses unsupported filter type ${type||'unknown'}.`);
  });
  return array?converted:converted[0];
}

function supportsFilter(parameter) {return /^(\[\])?(string|bool|u?int(8|16|32|64)?|float(32|64)?)$/.test(String(parameter.typeExpr||'').replace(/^\*/,''));}
