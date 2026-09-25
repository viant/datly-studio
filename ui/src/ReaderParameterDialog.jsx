import React, { useEffect, useMemo, useState } from 'react';
import { Button, ButtonGroup, Callout, Card, Dialog, DialogBody, DialogFooter, FormGroup, HTMLSelect, InputGroup, Tag } from '@blueprintjs/core';

const emptyInput = (mode='request') => ({ name: '', type: 'string', sourceKind: mode==='constant'?'const':'query', sourceName: '', required: false, querySelector: '', value: '', codecName: '', codecArgs: '', uri: '', emitOutput: false, description:'', example:'' });

export function ReaderParameterDialog({ isOpen, structure, initialName = '', initialMode = '', formOnly = false, embedded = false, onClose, onApply }) {
  const inputs = useMemo(() => editableInputs(structure), [structure]);
  const [draft, setDraft] = useState(emptyInput);
  const [existingName, setExistingName] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [removeName, setRemoveName] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [query, setQuery] = useState('');
  const [mode, setMode] = useState('request');
  const [kind, setKind] = useState('all');
  const [page, setPage] = useState(0);
  const pageSize = 50;
  const requestInputs=inputs.filter((input)=>String(input.source?.kind).toLowerCase()!=='const');
  const constants=inputs.filter((input)=>String(input.source?.kind).toLowerCase()==='const');
  const scoped=mode==='constant'?constants:requestInputs;
  const filtered = scoped.filter((input) => (kind === 'all' || String(input.source?.kind).toLowerCase() === kind) && `${input.name} ${input.typeExpr || ''} ${input.source?.kind || ''} ${input.source?.name || ''} ${input.codec?.body || ''} ${input.resourceRef || ''}`.toLowerCase().includes(query.trim().toLowerCase()));
  const pages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pages - 1);
  const visible = filtered.slice(currentPage * pageSize, currentPage * pageSize + pageSize);

  const beginAdd = (targetMode=mode) => { setDraft(emptyInput(targetMode)); setExistingName(''); setRemoveName(''); setConfirmation(''); };
  useEffect(() => { if (isOpen) { const initial = inputs.find((item) => item.name === initialName); const targetMode = initial ? String(initial.source?.kind).toLowerCase() === 'const' ? 'constant' : 'request' : initialMode || 'request'; setMode(targetMode); setDraft(initial ? inputDraft(initial) : emptyInput(targetMode)); setExistingName(initial?.name || ''); setRemoveName(''); setConfirmation(''); setSaving(false); setError(''); setQuery(''); setKind('all'); setPage(0); } }, [isOpen, initialName, initialMode]);
  useEffect(() => { if (!isOpen) return; setQuery(''); setKind('all'); setPage(0); }, [mode]);
  useEffect(() => { setPage(0); }, [query, kind]);
  const beginEdit = (input) => {
    setExistingName(input.name); setRemoveName(''); setConfirmation(''); setError('');
    setDraft(inputDraft(input));
  };
  const update = (name) => (event) => setDraft((current) => ({ ...current, [name]: event.target.value }));
  const submit = async (event) => {
    event.preventDefault();
    if (!/^[A-Za-z][A-Za-z0-9]*$/.test(draft.name) || !draft.sourceName.trim()) { setError('Use an UpperCamel input name and provide its source name.'); return; }
    setSaving(true); setError('');
    try {
      const field={name:draft.name,type:draft.type,sourceKind:draft.sourceKind,sourceName:draft.sourceName.trim(),required:draft.sourceKind==='const'?undefined:draft.required==='true'||draft.required===true,uri:draft.uri.trim()||null,updateUri:true,codec:draft.codecName.trim()?{name:draft.codecName.trim(),args:draft.codecArgs.split(',').map((item)=>item.trim()).filter(Boolean)}:null,updateCodec:true,emitOutput:draft.emitOutput===true,description:draft.description.trim()||null,updateDescription:true,example:draft.example===''?null:draft.example,updateExample:true};
      if (draft.sourceKind === 'const' && draft.value !== '') field.value = draft.value;
      if(existingName){field.existingName=existingName;field.updateQuerySelector=draft.sourceKind==='const'?'':draft.querySelector.trim();field.updateValue=true;if(draft.sourceKind!=='const'||draft.value==='')field.value=null;await onApply({type:'updateField',field});}
      else{field.querySelector=draft.querySelector.trim();await onApply({type:'addField',field});}
      beginAdd();
    } catch (cause) { setError(cause.message); }
    finally { setSaving(false); }
  };
  const remove = async () => {
    if(confirmation!==removeName)return;
    setSaving(true);setError('');
    try{await onApply({type:'removeField',field:{existingName:removeName}});beginAdd();}
    catch(cause){setError(cause.message);}finally{setSaving(false);}
  };

  if (!isOpen) return null;
  const content = <form onSubmit={submit} autoComplete="off">
      <DialogBody className="studio-connector-dialog-body">
        <p className="studio-dialog-lead">Manage typed request bindings and trusted constants. Request values remain bound parameters; constants never become browser-authored SQL interpolation.</p>
        {error && <Callout intent="danger" role="alert">{error}</Callout>}
        {!formOnly && <div className="studio-input-mode" role="tablist" aria-label="Input contract workspace"><Button type="button" small active={mode==='request'} role="tab" aria-selected={mode==='request'} onClick={()=>{setMode('request');beginAdd('request');}}>Request inputs <Tag minimal>{requestInputs.length}</Tag></Button><Button type="button" small active={mode==='constant'} role="tab" aria-selected={mode==='constant'} onClick={()=>{setMode('constant');beginAdd('constant');}}>Trusted constants <Tag minimal>{constants.length}</Tag></Button></div>}
        {mode==='constant'&&<Callout compact intent="primary" title="Deployment-owned values">Authored defaults remain in DQL. A selected server instance file may override them, including with zero, false, or an empty string. Invocation input can never supply these values.</Callout>}
        {!formOnly && <><div className="studio-input-filters"><InputGroup leftIcon="search" aria-label="Search inputs" placeholder={mode==='constant'?'Find constant, type, codec, or resource':'Find request input, source, selector, or codec'} value={query} onChange={(event)=>setQuery(event.target.value)} rightElement={query?<Button minimal icon="cross" aria-label="Clear input search" onClick={()=>setQuery('')}/>:undefined}/>{mode==='request'&&<HTMLSelect aria-label="Input source filter" value={kind} onChange={(event)=>setKind(event.target.value)}><option value="all">All request sources</option><option value="query">Query</option><option value="path">Path</option><option value="header">Header</option><option value="cookie">Cookie</option><option value="form">Form</option></HTMLSelect>}</div>
        <div className="studio-input-summary"><Tag minimal intent={mode==='constant'?'primary':'none'}>{scoped.length} {mode==='constant'?'constants':'request inputs'}</Tag><span>{filtered.length} shown</span></div>
        <div className="studio-parameter-list" aria-label="Reader inputs">{visible.length===0?<span className="studio-muted">No inputs match these filters.</span>:visible.map((input)=><button type="button" key={input.name} className={`studio-parameter-item ${existingName===input.name?'selected':''}`} onClick={()=>beginEdit(input)}><span><strong>{input.name}</strong><small>{input.source?.kind}/{input.source?.name}</small></span><Tag minimal intent={String(input.source?.kind).toLowerCase()==='const'?'primary':'none'}>{input.typeExpr||'inferred'}</Tag></button>)}</div>
        {filtered.length > pageSize && <div className="studio-column-pages"><span>{currentPage*pageSize+1}–{Math.min((currentPage+1)*pageSize,filtered.length)} of {filtered.length}</span><ButtonGroup minimal><Button type="button" small icon="chevron-left" aria-label="Previous input page" disabled={currentPage===0} onClick={()=>setPage(currentPage-1)}/><span>{currentPage+1} / {pages}</span><Button type="button" small icon="chevron-right" aria-label="Next input page" disabled={currentPage+1>=pages} onClick={()=>setPage(currentPage+1)}/></ButtonGroup></div>}</>}
        <div className="studio-panel-heading studio-parameter-form-heading"><div><h3>{existingName?`Edit ${existingName}`:mode==='constant'?'Add trusted constant':'Add request input'}</h3><p className="studio-muted">Rename is reference-aware and recompiles the complete reader atomically.</p></div>{existingName&&<ButtonGroup minimal><Button type="button" small icon="add" onClick={()=>beginAdd(mode)}>New</Button><Button type="button" small icon="trash" intent="danger" onClick={()=>{setRemoveName(existingName);setConfirmation('');}}>Remove</Button></ButtonGroup>}</div>
        <div className="studio-form-grid"><FormGroup label={mode==='constant'?'Constant name':'Input name'} labelFor="parameter-name" helperText={existingName?'Executable references are renamed; quoted literals are preserved.':'UpperCamel Go/DQL contract name.'} required><InputGroup id="parameter-name" value={draft.name} onChange={(event)=>setDraft((current)=>({...current,name:event.target.value,sourceName:mode==='constant'&&(!current.sourceName||current.sourceName===current.name)?event.target.value:current.sourceName}))} placeholder={mode==='constant'?'TenantID':'Limit'} autoFocus={!existingName} disabled={saving}/></FormGroup><FormGroup label="Type expression" labelFor="parameter-type" required><InputGroup id="parameter-type" list="parameter-type-options" value={draft.type} onChange={update('type')} disabled={saving}/><datalist id="parameter-type-options"><option value="string"/><option value="int"/><option value="int64"/><option value="bool"/><option value="[]string"/><option value="[]int"/></datalist></FormGroup></div>
        <div className="studio-form-grid"><FormGroup label="Source" labelFor="parameter-source" required>{mode==='constant'?<InputGroup id="parameter-source" value="Trusted instance constant" disabled/>:<HTMLSelect id="parameter-source" value={draft.sourceKind} onChange={update('sourceKind')} fill disabled={saving}><option value="query">Query</option><option value="path">Path</option><option value="header">Header</option><option value="cookie">Cookie</option><option value="form">Form</option></HTMLSelect>}</FormGroup><FormGroup label={mode==='constant'?'Instance key':'Source name'} labelFor="parameter-source-name" required><InputGroup id="parameter-source-name" value={draft.sourceName} onChange={update('sourceName')} placeholder={mode==='constant'?'TenantID':'limit'} disabled={saving}/></FormGroup></div>
        {draft.sourceKind === 'const' && <FormGroup label="Authored default" labelFor="parameter-value" helperText="Optional. A present instance-file value overrides this default, including zero, false, or an empty string."><InputGroup id="parameter-value" value={draft.value} onChange={update('value')} placeholder="7" disabled={saving}/></FormGroup>}
        {mode==='request'&&<div className="studio-form-grid"><FormGroup label="Required" labelFor="parameter-required"><HTMLSelect id="parameter-required" value={String(draft.required)} onChange={update('required')} fill disabled={saving}><option value="false">Optional</option><option value="true">Required</option></HTMLSelect></FormGroup><FormGroup label="Query selector view" labelFor="parameter-selector" helperText="Optional view identity for limit, offset, order, or projection selectors."><InputGroup id="parameter-selector" value={draft.querySelector} onChange={update('querySelector')} placeholder="Records" disabled={saving}/></FormGroup></div>}
        <div className="studio-column-authoring-section"><h3>Binding contract</h3><p>Optional Datly codec, resource or route activation, and output behavior. These remain structured Reader Builder operations.</p><div className="studio-form-grid"><FormGroup label="Codec" labelFor="parameter-codec" helperText="Registered codec name, such as CSV or JwtClaim."><InputGroup id="parameter-codec" value={draft.codecName} onChange={update('codecName')} placeholder="CSV" disabled={saving}/></FormGroup><FormGroup label="Codec arguments" labelFor="parameter-codec-args" helperText="Ordered comma-separated codec values."><InputGroup id="parameter-codec-args" value={draft.codecArgs} onChange={update('codecArgs')} placeholder="trim" disabled={saving||!draft.codecName}/></FormGroup></div><div className="studio-form-grid"><FormGroup label="Resource or route URI" labelFor="parameter-uri" helperText="A leading / activates this parameter only for that exact route or relative suffix; other values identify a named resource."><InputGroup id="parameter-uri" value={draft.uri} onChange={update('uri')} placeholder="/{id} or assets:ids.sql" disabled={saving}/></FormGroup><FormGroup label="Response emission" labelFor="parameter-output" helperText="Expose this binding in the generated output contract only when required."><HTMLSelect id="parameter-output" value={String(draft.emitOutput)} onChange={(event)=>setDraft((current)=>({...current,emitOutput:event.target.value==='true'}))} fill disabled={saving}><option value="false">Input only</option><option value="true">Emit in output</option></HTMLSelect></FormGroup></div></div>
        <div className="studio-column-authoring-section"><h3>Documentation & testing</h3><p>Descriptions and examples flow into generated HTTP and MCP schemas; they are metadata, never runtime defaults.</p><FormGroup label="Description" labelFor="parameter-description"><InputGroup id="parameter-description" value={draft.description} onChange={update('description')} placeholder="Maximum records returned" disabled={saving}/></FormGroup><FormGroup label="Example value" labelFor="parameter-example" helperText="Illustrative input shown in generated documentation and test tooling."><InputGroup id="parameter-example" value={draft.example} onChange={update('example')} placeholder="100" disabled={saving}/></FormGroup></div>
        {removeName&&<Callout intent="danger" title={`Remove ${removeName}?`}><p>This removes the declaration and recompiles the complete reader. Type the exact input name to enable removal.</p><InputGroup value={confirmation} onChange={(event)=>setConfirmation(event.target.value)} placeholder={removeName} disabled={saving}/><Button type="button" intent="danger" icon="trash" disabled={confirmation!==removeName} loading={saving} onClick={remove} style={{marginTop:10}}>Remove input</Button></Callout>}
      </DialogBody>
      <DialogFooter actions={<><Button onClick={onClose} disabled={saving}>{embedded ? 'Back to inputs' : 'Close'}</Button><Button type="submit" intent="primary" loading={saving}>{existingName?`Save ${mode==='constant'?'constant':'input'}`:mode==='constant'?'Add constant':'Add input'}</Button></>}/>
    </form>;
  return embedded ? <Card className="studio-card studio-columns-panel studio-inline-contract-form" elevation={0}>{content}</Card> : <Dialog className="studio-connector-dialog studio-parameter-dialog" isOpen={isOpen} onClose={onClose} title={formOnly ? existingName ? `${existingName} settings` : 'Input settings' : 'Inputs & constants'} icon="parameter" canOutsideClickClose={!saving}>{content}</Dialog>;
}

function editableInputs(structure){return (structure?.declarations??[]).map((item)=>item.parameter).filter((parameter)=>parameter&&!['output','component','view'].includes(String(parameter.source?.kind||'').toLowerCase()));}
function inputDraft(input){return { name: input.name, type: input.typeExpr || 'string', sourceKind: input.source?.kind || 'query', sourceName: input.source?.name || '', required: input.required === true, querySelector: input.querySelector?.view || input.querySelector || '', value: input.value ?? '', codecName: input.codec?.body || '', codecArgs: (input.codec?.args??[]).join(', '), uri: input.activation?.uri || input.resourceRef || '', emitOutput: input.emitOutput === true, description:input.description||'', example:input.example||'' };}
