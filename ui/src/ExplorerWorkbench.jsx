import React,{useEffect,useId,useRef,useState} from 'react';
import {useSignalEffect} from '@preact/signals-react';
import {Button,Callout,FormGroup,InputGroup,Spinner,Tag} from '@blueprintjs/core';
import {WorkspaceRuntimeSurface} from 'forge/components';
import Editor from 'forge/editor';
import {ResultPreview} from 'forge/result-preview';

// Studio supplies operation callbacks. Forge owns editing, tabs, panels and
// result rendering. Every run captures its source and never replaces a prior run.
export default function ExplorerWorkbench({getState,executeQuery,createComponent,addSubview}) {
 const id=useId(),sequence=useRef(0),counts=useRef({query:0,run:0}),alive=useRef(true),seen=useRef({initial:false,sql:0,table:0});
 const [scope,setScope]=useState({}),[tabs,setTabs]=useState([]),[selected,setSelected]=useState('');
 const [notice,setNotice]=useState('');
 const tabHost=useRef(null);
 const [tabScroll,setTabScroll]=useState({overflow:false,left:false,right:false});
 useEffect(()=>{
  const rail=tabHost.current?.querySelector('[role="tablist"]');if(!rail)return;
  const measure=()=>{
   const maximum=Math.max(0,rail.scrollWidth-rail.clientWidth);
   const next={overflow:maximum>1,left:rail.scrollLeft>1,right:rail.scrollLeft<maximum-1};
   setTabScroll(previous=>Object.keys(next).every(key=>next[key]===previous[key])?previous:next);
  };
  const reveal=()=>{
   if(!rail.clientWidth)return;
   const tab=rail.querySelector('[aria-selected="true"]');if(!tab)return;
   const bounds=rail.getBoundingClientRect(),item=(tab.closest('[data-workspace-tab-id]')||tab).getBoundingClientRect(),inset=rail.scrollWidth>rail.clientWidth?36:0;
   const delta=item.right>bounds.right-inset?item.right-(bounds.right-inset):item.left<bounds.left+inset?item.left-(bounds.left+inset):0;
   if(delta)rail.scrollBy?.({left:delta,behavior:'auto'});
  };
  rail.addEventListener('scroll',measure,{passive:true});
  const observer=new ResizeObserver(()=>{reveal();measure();});observer.observe(rail);
  for(const child of rail.children)observer.observe(child);
  reveal();measure();
  return()=>{rail.removeEventListener('scroll',measure);observer.disconnect();};
 },[tabs.length,selected,tabs.map(tab=>tab.status).join(',')]);
 const scrollTabs=direction=>{
  const rail=tabHost.current?.querySelector('[role="tablist"]');
  rail?.scrollBy({left:direction*Math.max(120,rail.clientWidth*.8),behavior:'smooth'});
 };
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;};},[]);
 const append=tab=>{setTabs(current=>[...current,tab]);setSelected(tab.windowId);};
 const newQuery=(source,kind='query')=>{
  if(!source.connectorName){setNotice('Select an active connector first.');return;}
  const number=++sequence.current;
  append({windowId:`sql-${number}`,number,kind,label:kind==='table'?source.detail?.table?.name||`Table ${number}`:`SQL ${++counts.current.query}`,
   connectorName:source.connectorName,driver:source.driver,catalogName:source.catalogName||'',schemaName:source.schemaName||'',
   detail:kind==='table'?structuredClone(source.detail):null,sql:kind==='table'?source.sql||'':'',timeoutSeconds:60,status:'draft'});
  setNotice('');
 };
 useSignalEffect(()=>{
  const next=getState?.();if(!next)return;
  setScope(next);
  if(!seen.current.initial&&next.connectorName){seen.current.initial=true;newQuery(next);}
  if(next.sqlRequest&&next.sqlRequest!==seen.current.sql){seen.current.sql=next.sqlRequest;newQuery(next);}
  if(next.tableRequest&&next.tableRequest!==seen.current.table){seen.current.table=next.tableRequest;newQuery(next,'table');}
 });
 const update=(tabId,changes)=>setTabs(current=>current.map(tab=>tab.windowId===tabId?{...tab,...changes}:tab));
 const close=tabId=>{
  const target=tabs.find(tab=>tab.windowId===tabId);if(target?.status==='running')return false;
  const remaining=tabs.filter(tab=>tab.windowId!==tabId);
  setTabs(remaining);if(selected===tabId)setSelected(remaining.at(-1)?.windowId||'');return true;
 };
 const run=async source=>{
  const timeoutSeconds=Number(source.timeoutSeconds);
  if(!source.sql.trim()){setNotice('Enter SQL to run.');return;}
  if(!Number.isSafeInteger(timeoutSeconds)||timeoutSeconds<0){setNotice('Timeout must be a nonnegative whole number of seconds. Use 0 for no Studio timeout.');return;}
  const number=++sequence.current,windowId=`run-${number}`,started=performance.now();
  const snapshot={...source,windowId,number,label:`Run ${++counts.current.run}`,kind:'execution',status:'running',timeoutSeconds,data:undefined,error:'',elapsed:0};
  append(snapshot);setNotice('');
  try{
   const result=await executeQuery({connectorName:snapshot.connectorName,sql:snapshot.sql,timeoutSeconds,limit:50});
   if(alive.current)update(windowId,{status:'complete',data:result?.data,elapsed:performance.now()-started});
  }catch(error){if(alive.current)update(windowId,{status:'failed',error:error.message||'Query execution failed.',elapsed:performance.now()-started});}
 };
 return <section className="studio-explorer-workbench" aria-label="SQL workspace">
  <div className="studio-explorer-query-header">
   <Button icon="add" onClick={()=>newQuery(scope)} disabled={!scope.connectorName}>New SQL tab</Button>
   <span>SQL and table tabs keep their own connector. Each run opens a result tab.</span>
  </div>
  {notice&&<Callout intent="warning" role="alert">{notice}</Callout>}
  {!tabs.length?<div className="studio-explorer-query-empty">Select a connector and open SQL or inspect a table.</div>:
   <div ref={tabHost} className={`studio-explorer-tab-shell${tabScroll.overflow?' has-overflow':''}`}>
    {tabScroll.overflow&&<>
     <Button className="studio-explorer-tab-scroll is-left" icon="chevron-left" aria-label="Scroll tabs left" title="Scroll tabs left" disabled={!tabScroll.left} onClick={()=>scrollTabs(-1)}/>
     <Button className="studio-explorer-tab-scroll is-right" icon="chevron-right" aria-label="Scroll tabs right" title="Scroll tabs right" disabled={!tabScroll.right} onClick={()=>scrollTabs(1)}/>
    </>}
   <WorkspaceRuntimeSurface className="studio-explorer-query-runtime"
    tabProps={{id,entries:tabs,selectedId:selected,onSelect:setSelected,onClose:close,canClose:tab=>tab.status!=='running',visibility:'always',ariaLabel:'SQL and table tabs',
     className:'studio-explorer-query-tabs',itemClassName:'studio-explorer-query-tab',tabClassName:'studio-explorer-query-tab-button',
     renderLabel:tab=><><span>{tab.label}</span>{' '}<span className="studio-explorer-tab-connector">{tab.connectorName}</span>{' '}{tab.kind==='execution'&&<span className={`studio-explorer-run-state is-${tab.status}`}>{tab.status==='complete'?'Done':tab.status==='failed'?'Failed':'Running'}</span>}</>}}
    panelProps={{id,entries:tabs,selectedId:selected,className:'studio-explorer-query-panel',renderContent:tab=><QueryPanel tab={tab} update={changes=>update(tab.windowId,changes)}
     run={()=>run(tab)} create={()=>createComponent({...tab})} addSubview={tab.kind==='table'?()=>addSubview({...tab}):null}/>}}/>
   </div>
  }
 </section>;
}

function QueryPanel({tab,update,run,create,addSubview}) {
 const timeoutID=useId(),[details,setDetails]=useState(false);
 const execution=tab.kind==='execution';
 return <>
  <div className="studio-explorer-query-context">
   <strong>{tab.connectorName}</strong><span>{[tab.driver,tab.catalogName,tab.schemaName].filter(Boolean).join(' · ')}</span>
   {execution&&<Tag minimal>{tab.status==='running'?'Running':`${tab.status==='complete'?'Completed':'Failed'} · ${(tab.elapsed/1000).toFixed(1)}s`}</Tag>}
  </div>
  <div className="studio-explorer-query-editor">
   <Editor value={tab.sql} onChange={sql=>update({sql})} language="sql" height="100%" readOnly={execution} ariaLabel={execution?'Executed SQL':'SQL query'}/>
  </div>
  <div className="studio-explorer-query-actions">
   <Button intent="primary" icon="play" onClick={run} disabled={!tab.sql.trim()||tab.status==='running'}>Run SQL</Button>
   <FormGroup label={execution?'Timeout used (seconds)':'Timeout (seconds)'} labelFor={timeoutID} className="studio-explorer-timeout">
    <InputGroup id={timeoutID} type="number" min={0} step={1} value={String(tab.timeoutSeconds)} readOnly={execution} onChange={event=>update({timeoutSeconds:event.target.value})}/>
   </FormGroup>
   <span className="studio-explorer-timeout-help">0 = no Studio timeout</span>
   <Button icon="new-object" onClick={create} disabled={!tab.sql.trim()||tab.status==='running'}>Create component</Button>
   {!execution&&tab.detail&&<Button icon="properties" active={details} onClick={()=>setDetails(value=>!value)}>Columns</Button>}
   {addSubview&&<Button icon="git-branch" onClick={addSubview}>Add as subview</Button>}
  </div>
  <div className="studio-explorer-query-results" aria-live="polite">
   {tab.status==='running'?<div className="studio-explorer-query-progress" role="status"><Spinner size={20}/> Running on {tab.connectorName} · {tab.timeoutSeconds===0?'No Studio timeout':`${tab.timeoutSeconds}s timeout`}</div>:
    tab.error?<Callout intent="danger" role="alert" title="Query failed">{tab.error}</Callout>:
    execution?<ResultPreview data={tab.data}/>:details?<ResultPreview data={{columns:tab.detail?.columns||[]}}/>:
    <div className="studio-explorer-query-empty">Run SQL to open a new execution tab. Results are limited to 50 rows; this does not limit the database scan.</div>}
  </div>
 </>;
}
