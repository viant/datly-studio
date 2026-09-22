import React, { useEffect, useState } from 'react';
import { Button, Callout, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Tag, TextArea } from '@blueprintjs/core';
import { LazyEditor as Editor } from './LazyEditor.jsx';
import { ResultPreview } from './NestedDataGrid.jsx';

// ReaderAnalyticsDialog exercises the saved cube-composition contract. Component
// toggles and budgets are authored cohesively in Component settings.
export function ReaderAnalyticsDialog({ isOpen, structure, onClose, onTestCompose }) {
  const report = structure?.component?.settings?.report;
  const cubeEnabled = Boolean(report?.enabled);
  const compose = report?.compose;
  const [error, setError] = useState('');
  const [frames, setFrames] = useState([]);
  const [composeSQL, setComposeSQL] = useState('SELECT * FROM $CubeSQL1');
  const [testing, setTesting] = useState(false);
  const [result, setResult] = useState(null);
  useEffect(() => {
    if (!isOpen) return;
    setTesting(false); setResult(null); setError('');
    setFrames([defaultFrameDraft(structure)]); setComposeSQL(defaultComposeSQL(structure));
  }, [isOpen, structure]);
  const testCompose = async () => {
    let cubes;
    try { cubes=frames.map(frameRequest); } catch(cause) { setError(cause.message); return; }
    if(cubes.length===0||!composeSQL.trim()){setError('Enter at least one cube frame and composition SQL.');return;}
    setTesting(true);setError('');setResult(null);
    try{setResult(await onTestCompose({cubes,sql:composeSQL.trim()}));}catch(cause){setError(cause.message);}finally{setTesting(false);}
  };
  const maxCubes = compose?.maxCubes ?? 8;
  return <Dialog className="studio-connector-dialog studio-compose-dialog" isOpen={isOpen} onClose={onClose} title="Composition lab" icon="heatmap" canOutsideClickClose={!testing}>
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">Exercise the saved Datly cube contract with ordered typed frames and composition SQL. Enable Cube and Cube composition from Component settings.</p>
        {error && <Callout intent="danger" role="alert" style={{ marginBottom: 14 }}>{error}</Callout>}
        {!cubeEnabled || !compose?.enabled ? <Callout intent="primary" title="Composition is not enabled">Open Edit component, enable Cube and Cube composition, then save the validated component revision.</Callout> : <section className="studio-compose-playground"><div className="studio-panel-heading"><div><h3>Composition playground</h3><p className="studio-muted">Build ordered typed cube frames, then author SQL only over $CubeSQL1…N and declared output aliases.</p></div><Tag minimal intent={frames.length<=Number(maxCubes)?'success':'danger'}>{frames.length} / {maxCubes} frames</Tag></div><div className="studio-compose-frames">{frames.map((frame,index)=><ComposeFrame key={frame.id} frame={frame} index={index} frames={frames} disabled={testing} onChange={(next)=>setFrames((current)=>current.map((item,itemIndex)=>itemIndex===index?next:item))} onRemove={()=>setFrames((current)=>current.filter((_,itemIndex)=>itemIndex!==index))}/>)}</div><Button type="button" icon="add" disabled={testing||frames.length>=Number(maxCubes)} onClick={()=>setFrames((current)=>[...current,emptyFrame(current.length)])}>Add cube frame</Button><FormGroup label="Composition SQL" labelFor="compose-sql" helperText="Only $CubeSQL1…N and validated output aliases are available."><div id="compose-sql" className="studio-code-editor"><Editor ariaLabel="Composition SQL source" value={composeSQL} onChange={setComposeSQL} language="sql" height="180px" readOnly={testing}/></div></FormGroup><Button type="button" icon="play" intent="primary" loading={testing} disabled={frames.length===0||frames.length>Number(maxCubes)} onClick={testCompose}>Run composition</Button>{result&&<div className="studio-compose-result"><ResultPreview data={result.data}/></div>}</section>}
      </DialogBody>
      <DialogFooter actions={<Button onClick={onClose} disabled={testing}>Close</Button>} />
  </Dialog>;
}

function defaultFrame(structure){const columns=structure?.component?.rootView?.columns??[];const dimensions={};const measures={};for(const column of columns){const name=lowerCamel(column.name||column.source);if(!name)continue;if(column.groupable===true||String(column.tag||'').includes('groupable:'))dimensions[name]=true;else measures[name]=true;}return {dimensions,measures,filters:{}};}
function defaultFrameDraft(structure){const frame=defaultFrame(structure);return{id:'frame-0',dimensions:Object.keys(frame.dimensions).join(', '),measures:Object.keys(frame.measures).join(', '),filters:'{}',inheritFrom:''};}
function emptyFrame(index){return{id:`frame-${Date.now()}-${index}`,dimensions:'',measures:'',filters:'{}',inheritFrom:''};}
function frameRequest(frame){let filters={};try{filters=JSON.parse(frame.filters||'{}');}catch{throw new Error('Every frame filter must be a JSON object.');}if(!filters||Array.isArray(filters)||typeof filters!=='object')throw new Error('Every frame filter must be a JSON object.');const dimensions=csvFlags(frame.dimensions);const measures=csvFlags(frame.measures);if(Object.keys(dimensions).length===0&&Object.keys(measures).length===0)throw new Error('Every cube frame needs at least one dimension or measure.');const result={dimensions,measures,filters};if(frame.inheritFrom!=='')result.inheritFrom=Number(frame.inheritFrom);return result;}
function csvFlags(value){const result={};for(const name of String(value||'').split(',').map((item)=>item.trim()).filter(Boolean))result[name]=true;return result;}
function defaultComposeSQL(structure){const columns=structure?.component?.rootView?.columns??[];const projections=columns.map((column)=>String(column.source||column.name||'').trim()).filter(Boolean).map((name)=>`t1.${name}`);return `SELECT ${projections.length?projections.join(', '):'t1.value'}\nFROM $CubeSQL1 AS t1`;}
function lowerCamel(value){const words=String(value??'').split(/[^A-Za-z0-9]+/).filter(Boolean).map((word)=>word.toLowerCase());return words.length?words[0]+words.slice(1).map((word)=>word[0].toUpperCase()+word.slice(1)).join(''):'';}

function ComposeFrame({frame,index,frames,disabled,onChange,onRemove}){const ordinal=index+1;return <section className="studio-compose-frame"><div className="studio-panel-heading"><div><h4>Frame {ordinal}</h4><code>{`$CubeSQL${ordinal}`}</code></div><Button type="button" small minimal icon="trash" intent="danger" aria-label={`Remove frame ${ordinal}`} disabled={disabled||frames.length===1} onClick={onRemove}/></div><div className="studio-form-grid"><FormGroup label="Dimensions" labelFor={`compose-dimensions-${ordinal}`}><InputGroup id={`compose-dimensions-${ordinal}`} aria-label={`Frame ${ordinal} dimensions`} value={frame.dimensions} onChange={(event)=>onChange({...frame,dimensions:event.target.value})} placeholder="status, region" disabled={disabled}/></FormGroup><FormGroup label="Measures" labelFor={`compose-measures-${ordinal}`}><InputGroup id={`compose-measures-${ordinal}`} aria-label={`Frame ${ordinal} measures`} value={frame.measures} onChange={(event)=>onChange({...frame,measures:event.target.value})} placeholder="productCount" disabled={disabled}/></FormGroup></div><FormGroup label="Inherit filters from" labelFor={`compose-inherit-${ordinal}`}><HTMLSelect id={`compose-inherit-${ordinal}`} aria-label={`Frame ${ordinal} inherited filters`} value={frame.inheritFrom} onChange={(event)=>onChange({...frame,inheritFrom:event.target.value})} fill disabled={disabled||index===0}><option value="">No inherited frame</option>{frames.slice(0,index).map((_,candidate)=><option key={candidate} value={candidate+1}>Frame {candidate+1}</option>)}</HTMLSelect></FormGroup><FormGroup label="Explicit filters" labelFor={`compose-filters-${ordinal}`} helperText="Typed cube filter object; omitted filters do not inherit unrelated request values."><TextArea id={`compose-filters-${ordinal}`} aria-label={`Frame ${ordinal} explicit filters`} value={frame.filters} onChange={(event)=>onChange({...frame,filters:event.target.value})} fill rows={2} className="studio-code-input" disabled={disabled}/></FormGroup></section>;}
