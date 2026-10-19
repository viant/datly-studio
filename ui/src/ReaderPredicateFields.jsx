import {fetchReaderDataSource} from './readerDataSources.js';
import React, {useEffect,useState} from 'react';
import {Button,Callout,FormGroup,HTMLSelect,InputGroup,NumericInput,TextArea} from '@blueprintjs/core';
import {readerViews,readerDimensions} from './readerViews.js';
import {predicateOptions,singleArgumentPredicate} from './predicateDraft.js';

const identity=(item)=>`${item.packagePath}.${item.typeName}`;
export function ReaderPredicateFields({api,structure,draft,onChange,disabled=false,editing=false,allowNone=false,idPrefix='predicate'}) {
 const [types,setTypes]=useState([]),[query,setQuery]=useState(''),[loading,setLoading]=useState(false),[error,setError]=useState(''),[attempt,setAttempt]=useState(0);
 const custom=draft.name==='handler';
 useEffect(()=>{
  if(!custom||disabled||!api)return;
  let cancelled=false;setLoading(true);setError('');
  const request=()=>fetchReaderDataSource(api,'predicateTypes');
  Promise.resolve().then(request).then((result)=>{if(!cancelled)setTypes(Array.isArray(result)?result:result?.items??[]);}).catch((cause)=>{if(!cancelled)setError(cause.message);}).finally(()=>{if(!cancelled)setLoading(false);});
  return()=>{cancelled=true;};
 },[api,custom,disabled,attempt]);
 const views=readerViews(structure);
 const names=predicateOptions(structure,draft.name);
 const update=(field)=>(event)=>onChange((current)=>({...current,[field]:event.target.value}));
 const selected=types.find((item)=>identity(item)===draft.args);
 const filtered=types.filter((item)=>[item.typeName,item.packagePath,item.alias,item.description].join(' ').toLowerCase().includes(query.toLowerCase()));
 const single=singleArgumentPredicate(draft.name);
 const dimensions=readerDimensions(structure,draft.view);
 const dimensionPredicate=['equal','in','not_equal','not_in','less_or_equal','less_than','greater_or_equal','greater_than','like','not_like','contains','not_contains','is_null','is_not_null'].includes(draft.name);
 const parts=String(draft.args||'').split(',').map((part)=>part.trim());
 const dimension=dimensions.find((item)=>item.name===`${parts[0]}.${parts[1]}`||item.name===parts[1]);
 const chooseDimension=(event)=>{const value=event.target.value;const qualified=/^([A-Za-z_][\w]*)\.([A-Za-z_][\w]*)$/.exec(value);onChange((current)=>({...current,args:qualified?`${qualified[1]}, ${qualified[2]}`:`${parts[0]||''}, ${value}`}));};
 const argumentField=<FormGroup label="Predicate arguments" labelFor={`${idPrefix}-args`} helperText={draft.name==='expr'?'One SQL expression; ? binds this input. Example: COALESCE(c.active, 0) = ?':single?'One expression argument, kept intact including commas.':'Ordered arguments: SQL alias, column for equal/in; use one argument per line for complex expressions.'} required>{single?<TextArea id={`${idPrefix}-args`} value={draft.args} onChange={update('args')} fill rows={2} disabled={disabled}/>:<InputGroup id={`${idPrefix}-args`} value={draft.args} onChange={update('args')} placeholder="c, id" disabled={disabled}/>}</FormGroup>;
 return <>
  <div className="studio-form-grid studio-binding-grid"><FormGroup label="Predicate" labelFor={`${idPrefix}-name`} required={!allowNone}><HTMLSelect id={`${idPrefix}-name`} value={draft.name} fill disabled={disabled} onChange={(event)=>{const name=event.target.value;onChange((current)=>({...current,name,args:name==='handler'||current.name==='handler'||singleArgumentPredicate(name)!==singleArgumentPredicate(current.name)?'':current.args}));}}>{allowNone&&<option value="">No predicate</option>}<optgroup label="Built-in predicates">{names.filter((name)=>name!=='handler').map((name)=><option key={name} value={name}>{name}</option>)}</optgroup><optgroup label="Custom predicates"><option value="handler">Custom Go predicate</option></optgroup></HTMLSelect></FormGroup>
  {draft.name&&<FormGroup label="Target view" labelFor={`${idPrefix}-view`} helperText={editing?'Existing expansion scope is preserved.':undefined} required><HTMLSelect id={`${idPrefix}-view`} value={draft.view} onChange={(event)=>onChange((current)=>({...current,view:event.target.value,args:dimensionPredicate?'':current.args}))} fill disabled={disabled||editing}>{views.map((view)=><option value={view.name} key={view.name}>{view.label}</option>)}</HTMLSelect></FormGroup>}</div>
  {draft.name&&<>
   {custom?<>
    <FormGroup label="Find custom predicate" labelFor={`${idPrefix}-search`}><InputGroup id={`${idPrefix}-search`} leftIcon="search" value={query} onChange={(event)=>setQuery(event.target.value)} placeholder="Search type, package, or description" disabled={disabled}/></FormGroup>
    {error&&<Callout intent="warning" title="Custom predicate discovery unavailable">{error}<Button small minimal onClick={()=>setAttempt((value)=>value+1)}>Retry discovery</Button></Callout>}
    <FormGroup label="Custom predicate" labelFor={`${idPrefix}-custom`} helperText={selected?.description||'Choose an actual handler linked by this host. No saved Security entry is required.'} required><HTMLSelect id={`${idPrefix}-custom`} value={draft.args} fill disabled={disabled||loading} onChange={(event)=>onChange((current)=>({...current,args:event.target.value}))}><option value="">{loading?'Loading discovered predicates…':'Select a discovered predicate'}</option>{draft.args&&!filtered.some((item)=>identity(item)===draft.args)&&<option value={draft.args}>{selected?`${selected.typeName} — ${selected.packagePath}`:`${draft.args} (current)`}</option>}{filtered.map((item)=><option key={identity(item)} value={identity(item)}>{item.typeName} — {item.packagePath}</option>)}</HTMLSelect></FormGroup>
    {!loading&&!error&&!filtered.length&&<p className="studio-muted">{query?'No discovered predicates match this search.':'This host has no discovered custom predicates.'}</p>}
    {draft.args&&<p className="studio-muted"><code>{draft.args}</code></p>}
   </>:<>{dimensionPredicate&&<div className="studio-form-grid studio-binding-grid"><FormGroup label="SQL alias" labelFor={`${idPrefix}-alias`}><InputGroup id={`${idPrefix}-alias`} value={parts[0]||''} disabled={disabled} onChange={(event)=>onChange((current)=>({...current,args:`${event.target.value}, ${parts[1]||''}`}))}/></FormGroup><FormGroup label="Filter dimension" labelFor={`${idPrefix}-dimension`} helperText={dimensions.length?'Columns discovered for the selected view.':'No columns discovered for this view.'}><HTMLSelect id={`${idPrefix}-dimension`} fill disabled={disabled||!dimensions.length} value={dimension?.name||''} onChange={chooseDimension}><option value="">Select a dimension</option>{dimensions.map((item)=><option key={item.name} value={item.name}>{item.label}</option>)}</HTMLSelect></FormGroup></div>}{dimensionPredicate?<details className="studio-predicate-arguments"><summary>Advanced arguments</summary>{argumentField}</details>:argumentField}</>}
   <details className="studio-column-authoring-section studio-predicate-options"><summary>Predicate options</summary><div className="studio-form-grid"><FormGroup label="Group" labelFor={`${idPrefix}-group`}><NumericInput id={`${idPrefix}-group`} value={draft.group} min={0} fill disabled={disabled} onValueChange={(group)=>onChange((current)=>({...current,group}))}/></FormGroup><FormGroup label="Within-group operator" labelFor={`${idPrefix}-operator`}><HTMLSelect id={`${idPrefix}-operator`} value={draft.groupOperator} onChange={update('groupOperator')} fill disabled={disabled||editing}><option value="AND">AND</option><option value="OR">OR</option></HTMLSelect></FormGroup></div><FormGroup label="When input is absent" labelFor={`${idPrefix}-absent`}><HTMLSelect id={`${idPrefix}-absent`} value={String(draft.applyWhenAbsent)} onChange={(event)=>onChange((current)=>({...current,applyWhenAbsent:event.target.value==='true'}))} disabled={disabled}><option value="false">Skip predicate</option><option value="true">Apply predicate</option></HTMLSelect></FormGroup></details>
  </>}
 </>;
}
